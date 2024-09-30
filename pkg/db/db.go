package db

import (
	"context"
	"errors"
	"fmt"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/multierr"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/Seann-Moser/QueryHelper"
	"github.com/XSAM/otelsql"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
)

type DAO struct {
	db            QueryHelper.DB
	updateColumns bool
	ctx           context.Context
	tablesNames   []string
	tableColumns  map[string]map[string]QueryHelper.Column
}

func NewMockDAO() *DAO {
	return &DAO{
		db:            QueryHelper.NewMockDB(),
		updateColumns: false,
		tablesNames:   make([]string, 0),
		tableColumns:  map[string]map[string]QueryHelper.Column{},
	}
}

const (
	DBUserNameFlag           = "db-user"
	DBPasswordFlag           = "db-password"
	DBHostFlag               = "db-host"
	DBPortFlag               = "db-port"
	DBMaxConnectionsFlag     = "db-max-connections"
	DBUpdateTablesFlag       = "db-update-table"
	DBMaxConnectionRetryFlag = "db-max-connection-retry"
	DBInstanceName           = "db-instance-name"
	DBWriteStatDuration      = "db-write-stat-interval"
	DBMaxIdleConnectionsFlag = "db-max-idle-connections-flag"
	DBMaxConnectionLifetime  = "db-max-connection-lifetime"
)

func GetDaoFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("db-dao", pflag.ExitOnError)
	fs.AddFlagSet(QueryHelper.Flags())
	fs.String(DBUserNameFlag, "", "")
	fs.String(DBPasswordFlag, "", "")
	fs.String(DBHostFlag, "mysql", "")
	fs.String(DBInstanceName, "resource", "ocsql instance name")

	fs.Int(DBPortFlag, 3306, "")
	fs.Int(DBMaxConnectionsFlag, 10, "")
	fs.Int(DBMaxIdleConnectionsFlag, 10, "")
	fs.Int(DBMaxConnectionRetryFlag, 10, "")
	fs.Bool(DBUpdateTablesFlag, false, "")
	fs.Duration(DBMaxConnectionLifetime, 1*time.Minute, "")
	fs.Duration(DBWriteStatDuration, 10*time.Second, "")

	return fs
}

func (d *DAO) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d.ctx != nil {
			daoCtx := context.WithValue(r.Context(), "go-serve-dao", d) //nolint staticcheck
			ctx, err := QueryHelper.WithTableContext(daoCtx, d.ctx, d.tablesNames...)
			if err == nil {
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (d *DAO) Close() {
	d.db.Close()

}

func (d *DAO) Ping(ctx context.Context) bool {
	return d.db.Ping(ctx) == nil
}
func (d *DAO) GetContext() context.Context {
	return d.ctx
}

func (d *DAO) AddTablesToCtx(ctx context.Context) context.Context {
	if d.ctx != nil {
		daoCtx := context.WithValue(ctx, "go-serve-dao", d) //nolint staticcheck
		ctx, err := QueryHelper.WithTableContext(daoCtx, d.ctx, d.tablesNames...)
		if err == nil {
			return ctx
		}
	}
	return d.ctx
}

func GetDao(ctx context.Context) (*DAO, error) {
	value := ctx.Value("go-serve-dao")
	if value == nil {
		return nil, errors.New("unable to get dao from context")
	}
	return value.(*DAO), nil
}

func DeleteAll[T any](ctx context.Context, i T) error {
	table, err := QueryHelper.GetTableCtx[T](ctx)
	if err != nil {
		return err
	}
	dao, err := GetDao(ctx)
	if err != nil {
		return err
	}
	for tableName, columns := range dao.tableColumns {
		tmpErr := table.DeleteWithColumns(ctx, tableName, columns, i)
		if err != nil && !strings.Contains(tmpErr.Error(), "could not find") {
			err = multierr.Combine(err, tmpErr)
		}
	}

	return err
}

//func (d *DAO) ContextWithTransaction(ctx context.Context) (context.Context, error) {
//	tx, err := d.db.BeginTxx(ctx, &sql.TxOptions{})
//	if err != nil {
//		return ctx, err
//	}
//	return context.WithValue(ctx, "transaction", tx), nil
//}

func ContextGetTransaction(ctx context.Context) (*sqlx.Tx, error) {
	value := ctx.Value("transaction")
	if value == nil {
		return nil, errors.New("no valid transaction in context")
	}
	if v, found := value.(*sqlx.Tx); found {
		return v, nil
	}
	return nil, errors.New("unable to type cast transaction")
}

func NewSQLDao(ctx context.Context) (*DAO, error) {
	db, err := connectToDB(
		ctx,
		viper.GetString(DBUserNameFlag),
		viper.GetString(DBPasswordFlag),
		viper.GetString(DBHostFlag),
		viper.GetString(DBInstanceName),
		viper.GetInt(DBPortFlag),
		viper.GetInt(DBMaxConnectionsFlag),
		viper.GetInt(DBMaxIdleConnectionsFlag),
		viper.GetDuration(DBMaxConnectionLifetime),
		viper.GetDuration(DBWriteStatDuration),
	)
	if err != nil {
		return nil, err
	}

	return &DAO{
		db:            QueryHelper.NewSql(db),
		updateColumns: viper.GetBool(DBUpdateTablesFlag),
		tablesNames:   make([]string, 0),
		tableColumns:  map[string]map[string]QueryHelper.Column{},
	}, nil
}

func AddTable[T any](ctx context.Context, dao *DAO, datasetName string, queryType QueryHelper.QueryType) (context.Context, error) {
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

func getType(myVar interface{}) string {
	if t := reflect.TypeOf(myVar); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}

func connectToDB(ctx context.Context, user, password, host, instanceName string, port, maxConnections, idleConn int, lifeTime, writeStatDuration time.Duration) (*sqlx.DB, error) {
	dbConf := mysql.Config{
		AllowNativePasswords:    true,
		User:                    user,
		Passwd:                  password,
		Net:                     "tcp",
		Addr:                    fmt.Sprintf("%s:%d", host, port),
		CheckConnLiveness:       true,
		AllowCleartextPasswords: true,
		MaxAllowedPacket:        4 << 20,
	}
	ctxLogger.Info(ctx, "connecting to db", zap.String("dsn", dbConf.FormatDSN()))

	otelSql, err := otelsql.Open("mysql", dbConf.FormatDSN(), otelsql.WithAttributes(
		semconv.DBSystemMySQL))
	if err != nil {
		return nil, err
	}
	db := sqlx.NewDb(otelSql, "mysql")
	db.SetMaxOpenConns(maxConnections)
	db.SetConnMaxLifetime(lifeTime)
	db.SetMaxIdleConns(idleConn)
	db.SetConnMaxIdleTime(10 * time.Minute)
	if err = db.Ping(); err == nil {
		return db, nil
	}
	var retries int
	ticker := time.NewTicker(5 * time.Second)
	err = otelsql.RegisterDBStatsMetrics(otelSql, otelsql.WithAttributes(
		semconv.DBSystemMySQL,
	))
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled")
		case <-ticker.C:
			if retries >= viper.GetInt(DBMaxConnectionRetryFlag) {
				return nil, err
			}
			if err = db.Ping(); err == nil {
				return db, nil
			}
			ctxLogger.Info(ctx, "attempting to connect to db", zap.Int("attempt", retries), zap.String("dsn", dbConf.FormatDSN()))
			retries++
		}
	}

}
