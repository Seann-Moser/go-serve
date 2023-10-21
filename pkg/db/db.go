package db

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/QueryHelper"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"net/http"
	"reflect"
	"time"
)

type DAO struct {
	db            *sqlx.DB
	dropTable     bool
	updateColumns bool
	ctx           context.Context
}

const (
	DBUserNameFlag           = "db-user"
	DBPasswordFlag           = "db-password"
	DBHostFlag               = "db-host"
	DBPortFlag               = "db-port"
	DBMaxConnectionsFlag     = "db-max-connections"
	DBUpdateTablesFlag       = "db-update-table"
	DBMaxConnectionRetryFlag = "db-max-connection-retry"
)

func GetDaoFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("db-dao", pflag.ExitOnError)
	fs.String(DBUserNameFlag, "", "")
	fs.String(DBPasswordFlag, "", "")
	fs.String(DBHostFlag, "mysql", "")
	fs.Int(DBPortFlag, 3306, "")
	fs.Int(DBMaxConnectionsFlag, 10, "")
	fs.Int(DBMaxConnectionRetryFlag, 10, "")
	fs.Bool(DBUpdateTablesFlag, false, "")

	return fs
}

func (d *DAO) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d.ctx != nil {
			r = r.WithContext(d.ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func NewDao(ctx context.Context) (*DAO, error) {
	db, err := connectToDB(
		ctx,
		viper.GetString(DBUserNameFlag),
		viper.GetString(DBPasswordFlag),
		viper.GetString(DBHostFlag),
		viper.GetInt(DBPortFlag),
		viper.GetInt(DBMaxConnectionsFlag))
	if err != nil {
		return nil, err
	}
	return &DAO{db: db, updateColumns: viper.GetBool(DBUpdateTablesFlag)}, nil
}

func AddTable[T any](ctx context.Context, dao *DAO, datasetName string) (context.Context, error) {
	tmpCtx, err := QueryHelper.AddTableCtx[T](ctx, dao.db, datasetName, dao.dropTable, dao.updateColumns)
	if err != nil {
		var t T
		ctxLogger.Error(ctx, "failed creating table", zap.String("table", getType(t)))
		return nil, err
	}
	table, err := QueryHelper.GetTableCtx[T](tmpCtx)
	if err != nil {
		return nil, err
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
func connectToDB(ctx context.Context, user, password, host string, port, maxConnections int) (*sqlx.DB, error) {
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

	db, err := sqlx.Open("mysql", dbConf.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxConnections)
	if err = db.Ping(); err == nil {
		return db, nil
	}
	var retries int
	ticker := time.NewTicker(5 * time.Second)
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
