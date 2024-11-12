package db

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/Seann-Moser/QueryHelper"
	"github.com/XSAM/otelsql"
	"github.com/cenkalti/backoff/v4"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
)

// DAO represents the Data Access Object, providing methods to interact with the database.
type DAO struct {
	db            QueryHelper.DB
	updateColumns bool
	ctx           context.Context
	tablesNames   []string
	tableColumns  map[string]map[string]QueryHelper.Column
}

// NewMockDAO creates a new DAO instance with a mock database.
func NewMockDAO() *DAO {
	return &DAO{
		db:            QueryHelper.NewMockDB(),
		updateColumns: false,
		tablesNames:   make([]string, 0),
		tableColumns:  map[string]map[string]QueryHelper.Column{},
	}
}

const (
	DBUserNameFlag                   = "db-user"
	DBPasswordFlag                   = "db-password"
	DBHostFlag                       = "db-host"
	DBPortFlag                       = "db-port"
	DBMaxConnectionsFlag             = "db-max-connections"
	DBType                           = "db-type"
	DBUpdateTablesFlag               = "db-update-table"
	DBMaxConnectionRetryFlag         = "db-max-connection-retry"
	DBMaxConnectionRetryDurationFlag = "db-max-connection-retry-duration"
	DBPingTimeoutFlag                = "db-ping-timeout"
	DBInstanceName                   = "db-instance-name"
	DBWriteStatDuration              = "db-write-stat-interval"
	DBMaxIdleConnectionsFlag         = "db-max-idle-connections"
	DBMaxConnectionLifetime          = "db-max-connection-lifetime"
	DBMaxIdleTimeFlag                = "db-max-idle-time"
)

// GetDaoFlags returns the command-line flags for database configuration.
func GetDaoFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("db-dao", pflag.ExitOnError)
	fs.AddFlagSet(QueryHelper.Flags())
	fs.String(DBUserNameFlag, "", "Database username")
	fs.String(DBPasswordFlag, "", "Database password")
	fs.String(DBHostFlag, "mysql", "Database host")
	fs.String(DBInstanceName, "resource", "Database instance name")

	fs.Int(DBPortFlag, 3306, "Database port")
	fs.Int(DBMaxConnectionsFlag, 10, "Maximum open database connections")
	fs.Int(DBMaxIdleConnectionsFlag, 10, "Maximum idle database connections")
	fs.Int(DBMaxConnectionRetryFlag, 10, "Maximum database connection retries")
	fs.Duration(DBMaxConnectionRetryDurationFlag, 1*time.Minute, "Maximum duration for connection retries")
	fs.Duration(DBPingTimeoutFlag, 5*time.Second, "Timeout for pinging the database")
	fs.Bool(DBUpdateTablesFlag, false, "Update database tables")
	fs.String(DBType, "mysql", "Database type (mysql, postgres, sqlite)")
	fs.Duration(DBMaxConnectionLifetime, 1*time.Minute, "Maximum database connection lifetime")
	fs.Duration(DBWriteStatDuration, 10*time.Second, "Interval for writing database stats")
	fs.Duration(DBMaxIdleTimeFlag, 10*time.Minute, "Maximum idle time for database connections")

	return fs
}

// Middleware injects the DAO into the request context.
func (d *DAO) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d.ctx != nil {
			daoCtx := context.WithValue(r.Context(), "go-serve-dao", d) //nolint:staticcheck
			ctx, err := QueryHelper.WithTableContext(daoCtx, d.ctx, d.tablesNames...)
			if err == nil {
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Close closes the database connection.
func (d *DAO) Close() {
	d.db.Close()
}

// Ping checks the database connectivity.
func (d *DAO) Ping(ctx context.Context) bool {
	backoffPolicy := backoff.NewExponentialBackOff()
	backoffPolicy.MaxElapsedTime = 10 * time.Second

	var dbErr error
	err := backoff.Retry(func() error {
		pingCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		dbErr = d.db.Ping(pingCtx)
		if dbErr != nil {
			if isNonRetryableError(dbErr) {
				return backoff.Permanent(dbErr)
			}
			ctxLogger.Warn(ctx, "failed to ping db", zap.Error(dbErr))
			return dbErr
		}
		return nil
	}, backoffPolicy)
	if err != nil {
		ctxLogger.Error(ctx, "failed to ping db", zap.Error(dbErr))
		return false
	}
	return true
}

// GetContext returns the DAO's context.
func (d *DAO) GetContext() context.Context {
	return d.ctx
}

// AddTablesToCtx adds table information to the context.
func (d *DAO) AddTablesToCtx(ctx context.Context) context.Context {
	if d.ctx != nil {
		daoCtx := context.WithValue(ctx, "go-serve-dao", d) //nolint:staticcheck
		ctx, err := QueryHelper.WithTableContext(daoCtx, d.ctx, d.tablesNames...)
		if err == nil {
			return ctx
		}
	}
	return d.ctx
}

// GetDao retrieves the DAO from the context.
func GetDao(ctx context.Context) (*DAO, error) {
	value := ctx.Value("go-serve-dao")
	if value == nil {
		return nil, errors.New("unable to get dao from context")
	}
	dao, ok := value.(*DAO)
	if !ok {
		return nil, errors.New("invalid dao type in context")
	}
	return dao, nil
}

// DeleteAll deletes all records of type T.
func DeleteAll[T any](ctx context.Context, i T) error {
	table, err := QueryHelper.GetTableCtx[T](ctx)
	if err != nil {
		return err
	}
	dao, err := GetDao(ctx)
	if err != nil {
		return err
	}
	var combinedErr error
	for tableName, columns := range dao.tableColumns {
		tmpErr := table.DeleteWithColumns(ctx, tableName, columns, i)
		if tmpErr != nil && !strings.Contains(tmpErr.Error(), "could not find") {
			combinedErr = multierr.Append(combinedErr, tmpErr)
		}
	}
	return combinedErr
}

// ContextGetTransaction retrieves the transaction from the context.
func ContextGetTransaction(ctx context.Context) (*sqlx.Tx, error) {
	value := ctx.Value("transaction")
	if value == nil {
		return nil, errors.New("no valid transaction in context")
	}
	tx, ok := value.(*sqlx.Tx)
	if !ok {
		return nil, errors.New("unable to type cast transaction")
	}
	return tx, nil
}

// NewSQLDao creates a new DAO with a real SQL database connection.
func NewSQLDao(ctx context.Context) (*DAO, error) {
	cfg := DBConfig{
		MaxOpenConns:               viper.GetInt(DBMaxConnectionsFlag),
		MaxIdleConns:               viper.GetInt(DBMaxIdleConnectionsFlag),
		ConnMaxLifetime:            viper.GetDuration(DBMaxConnectionLifetime),
		MaxIdleTime:                viper.GetDuration(DBMaxIdleTimeFlag),
		MaxConnectionRetries:       viper.GetInt(DBMaxConnectionRetryFlag),
		MaxConnectionRetryDuration: viper.GetDuration(DBMaxConnectionRetryDurationFlag),
		PingTimeout:                viper.GetDuration(DBPingTimeoutFlag),
	}

	db, err := connectToDB(
		ctx,
		cfg,
		viper.GetString(DBType),
		viper.GetString(DBUserNameFlag),
		viper.GetString(DBPasswordFlag),
		viper.GetString(DBHostFlag),
		viper.GetString(DBInstanceName),
		viper.GetInt(DBPortFlag),
	)
	if err != nil {
		return nil, err
	}
	d := QueryHelper.NewSql(db)
	QueryHelper.AddDBContext(ctx, "", d)
	return &DAO{
		db:            d,
		updateColumns: viper.GetBool(DBUpdateTablesFlag),
		tablesNames:   make([]string, 0),
		tableColumns:  map[string]map[string]QueryHelper.Column{},
		ctx:           QueryHelper.AddDBContext(ctx, "", d),
	}, nil
}

// AddTable adds a table of type T to the DAO.
func AddTable[T any](ctx context.Context, dao *DAO, datasetName string, queryType QueryHelper.QueryType) (context.Context, error) {
	if dao.ctx != nil {
		ctx = dao.ctx
	}
	tmpCtx, err := QueryHelper.AddTableCtx[T](ctx, dao.db, datasetName, queryType)
	if err != nil {
		var t T
		ctxLogger.Error(ctx, "failed creating table", zap.String("table", getType(t)))
		return ctx, err
	}

	table, err := QueryHelper.GetTableCtx[T](tmpCtx)
	if err != nil {
		return nil, err
	}
	dao.tablesNames = append(dao.tablesNames, table.Name)
	if _, found := dao.tableColumns[table.FullTableName()]; !found {
		dao.tableColumns[table.FullTableName()] = table.Columns
	}

	ctxLogger.Debug(ctx, "adding table", zap.String("table", table.FullTableName()))
	dao.ctx = tmpCtx
	return tmpCtx, nil
}

// getType returns the type name of the variable.
func getType(myVar interface{}) string {
	t := reflect.TypeOf(myVar)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// DBConfig holds the database configuration parameters.
type DBConfig struct {
	MaxOpenConns               int
	MaxIdleConns               int
	ConnMaxLifetime            time.Duration
	MaxIdleTime                time.Duration
	MaxConnectionRetries       int
	MaxConnectionRetryDuration time.Duration
	PingTimeout                time.Duration
}

// connectToDB establishes a database connection with retry logic.
func connectToDB(ctx context.Context, cfg DBConfig, dbType, user, password, host, instanceName string, port int) (*sqlx.DB, error) {
	var dsn string
	var dbSystem attribute.KeyValue

	switch dbType {
	case "mysql":
		mysqlConf := mysql.Config{
			AllowNativePasswords:    true,
			User:                    user,
			Passwd:                  password,
			Net:                     "tcp",
			Addr:                    fmt.Sprintf("%s:%d", host, port),
			CheckConnLiveness:       true,
			AllowCleartextPasswords: true,
			MaxAllowedPacket:        4 << 20,
		}
		dsn = mysqlConf.FormatDSN()
		dbSystem = semconv.DBSystemMySQL

	case "postgres":
		dbSystem = semconv.DBSystemPostgreSQL
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, instanceName)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	ctxLogger.Debug(ctx, "connecting to db",
		zap.String("dbType", dbType),
		zap.String("host", host),
		zap.Int("port", port),
		zap.String("instanceName", instanceName),
	)

	otelSql, err := otelsql.Open(dbType, dsn, otelsql.WithAttributes(
		dbSystem,
	))
	if err != nil {
		return nil, err
	}

	db := sqlx.NewDb(otelSql, dbType)
	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxIdleTime(cfg.MaxIdleTime)

	if err = otelsql.RegisterDBStatsMetrics(otelSql, otelsql.WithAttributes(
		dbSystem,
	)); err != nil {
		ctxLogger.Error(ctx, "failed to register db stats metrics", zap.Error(err))
		// Decide whether to continue or return an error
	}

	backoffPolicy := backoff.NewExponentialBackOff()
	backoffPolicy.MaxElapsedTime = cfg.MaxConnectionRetryDuration

	var dbErr error
	err = backoff.Retry(func() error {
		pingCtx, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
		defer cancel()

		dbErr = db.PingContext(pingCtx)
		if dbErr != nil {
			if isNonRetryableError(dbErr) {
				return backoff.Permanent(dbErr)
			}
			ctxLogger.Warn(ctx, "failed to ping db", zap.Error(dbErr))
			return dbErr
		}
		return nil
	}, backoffPolicy)

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to db after retries: %w", dbErr)
	}

	return db, nil
}

// isNonRetryableError determines if the error is non-retryable.
func isNonRetryableError(err error) bool {
	// Implement logic to determine if an error is non-retryable
	// For example, authentication errors, invalid configuration, etc.
	// For this example, we'll assume all errors are retryable
	// In a real implementation, inspect err and return true if it's non-retryable
	return false
}
