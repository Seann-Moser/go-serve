package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/opencensus-integrations/ocsql"
	"net/http"
	"reflect"
	"time"

	"github.com/Seann-Moser/QueryHelper"
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
)

func GetDaoFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("db-dao", pflag.ExitOnError)
	fs.String(DBUserNameFlag, "", "")
	fs.String(DBPasswordFlag, "", "")
	fs.String(DBHostFlag, "mysql", "")
	fs.String(DBInstanceName, "resource", "ocsql instance name")

	fs.Int(DBPortFlag, 3306, "")
	fs.Int(DBMaxConnectionsFlag, 10, "")
	fs.Int(DBMaxConnectionRetryFlag, 10, "")
	fs.Bool(DBUpdateTablesFlag, false, "")
	fs.Duration(DBWriteStatDuration, 30*time.Second, "")

	return fs
}

func (d *DAO) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d.ctx != nil {
			ctx, err := QueryHelper.WithTableContext(r.Context(), d.ctx, d.tablesNames...)
			if err == nil {
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}
func (d *DAO) GetContext() context.Context {
	return d.ctx
}

func (d *DAO) AddTablesToCtx(ctx context.Context) context.Context {
	if d.ctx != nil {
		ctx, err := QueryHelper.WithTableContext(ctx, d.ctx, d.tablesNames...)
		if err == nil {
			return ctx
		}
	}
	return d.ctx
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
		viper.GetDuration(DBWriteStatDuration),
	)
	if err != nil {
		return nil, err
	}

	return &DAO{db: QueryHelper.NewSql(db), updateColumns: viper.GetBool(DBUpdateTablesFlag), tablesNames: make([]string, 0)}, nil
}

func AddTable[T any](ctx context.Context, dao *DAO, datasetName string, queryType QueryHelper.QueryType) (context.Context, error) {
	tmpCtx, err := QueryHelper.AddTableCtx[T](ctx, dao.db, datasetName, queryType)
	if err != nil {
		var t T
		ctxLogger.Error(ctx, "failed creating table", zap.String("table", getType(t)))
		return nil, err
	}
	table, err := QueryHelper.GetTableCtx[T](tmpCtx)
	if err != nil {
		return nil, err
	}
	dao.tablesNames = append(dao.tablesNames, table.Name)
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
func connectToDB(ctx context.Context, user, password, host, instanceName string, port, maxConnections int, writeStatDuration time.Duration) (*sqlx.DB, error) {
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
	//dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, host, port)
	ctxLogger.Info(ctx, "connecting to db", zap.String("dsn", dbConf.FormatDSN()))
	driverName, err := ocsql.Register("mysql", ocsql.WithAllTraceOptions(), ocsql.WithInstanceName(instanceName))
	if err != nil {
		return nil, err
	}
	sqlDb, err := sql.Open(driverName, dbConf.FormatDSN())
	if err != nil {
		return nil, err
	}
	sqlDb.SetMaxOpenConns(maxConnections)

	sqlx.NewDb(sqlDb, "mysql")
	db := sqlx.NewDb(sqlDb, "mysql")

	if err = db.Ping(); err == nil {
		return db, nil
	}
	var retries int
	ticker := time.NewTicker(5 * time.Second)
	if writeStatDuration != 0 {
		defer ocsql.RecordStats(sqlDb, writeStatDuration)()
	}

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
