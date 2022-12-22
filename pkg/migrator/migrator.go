package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/NpoolPlatform/ledger-manager/pkg/db"
	"github.com/NpoolPlatform/ledger-manager/pkg/db/ent"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	entaccount "github.com/NpoolPlatform/account-manager/pkg/db/ent"
	accountconst "github.com/NpoolPlatform/account-manager/pkg/message/const"
	entbilling "github.com/NpoolPlatform/cloud-hashing-billing/pkg/db/ent"
	billingconst "github.com/NpoolPlatform/cloud-hashing-billing/pkg/message/const"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	constant "github.com/NpoolPlatform/go-service-framework/pkg/mysql/const"

	_ "github.com/NpoolPlatform/account-manager/pkg/db/ent/runtime"
	_ "github.com/NpoolPlatform/review-service/pkg/db/ent/runtime"
)

const (
	keyUsername = "username"
	keyPassword = "password"
	keyDBName   = "database_name"
	maxOpen     = 10
	maxIdle     = 10
	MaxLife     = 3
)

func dsn(hostname string) (string, error) {
	username := config.GetStringValueWithNameSpace(constant.MysqlServiceName, keyUsername)
	password := config.GetStringValueWithNameSpace(constant.MysqlServiceName, keyPassword)
	dbname := config.GetStringValueWithNameSpace(hostname, keyDBName)

	svc, err := config.PeekService(constant.MysqlServiceName)
	if err != nil {
		logger.Sugar().Warnw("dsb", "error", err)
		return "", err
	}

	return fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?parseTime=true&interpolateParams=true",
		username, password,
		svc.Address,
		svc.Port,
		dbname,
	), nil
}

func open(hostname string) (conn *sql.DB, err error) {
	hdsn, err := dsn(hostname)
	if err != nil {
		return nil, err
	}

	logger.Sugar().Infow("open", "hdsn", hdsn)

	conn, err = sql.Open("mysql", hdsn)
	if err != nil {
		return nil, err
	}

	// https://github.com/go-sql-driver/mysql
	// See "Important settings" section.

	conn.SetConnMaxLifetime(time.Minute * MaxLife)
	conn.SetMaxOpenConns(maxOpen)
	conn.SetMaxIdleConns(maxIdle)

	return conn, nil
}

func migrateWithdrawAddress(ctx context.Context) error {
	billing, err := open(billingconst.ServiceName)
	if err != nil {
		logger.Sugar().Errorw("migrateWithdrawAddress", "error", err)
		return err
	}
	defer billing.Close()

	bcli := entbilling.NewClient(entbilling.Driver(entsql.OpenDB(dialect.MySQL, billing)))
	baccounts, err := bcli.
		CoinAccountInfo.
		Query().
		All(ctx)
	if err != nil {
		logger.Sugar().Errorw("migrateWithdrawAddress", "error", err)
		return err
	}

	account, err := open(accountconst.ServiceName)
	if err != nil {
		logger.Sugar().Errorw("migrateWithdrawAddress", "error", err)
		return err
	}
	defer account.Close()

	acli := entaccount.NewClient(entaccount.Driver(entsql.OpenDB(dialect.MySQL, account)))
	aaccounts, err := acli.
		Account.
		Query().
		All(ctx)
	if err != nil {
		logger.Sugar().Errorw("migrateReview", "error", err)
		return err
	}

	return db.WithClient(ctx, func(_ctx context.Context, cli *ent.Client) error {
		withdraws, err := cli.
			Withdraw.
			Query().
			All(_ctx)
		if err != nil {
			return err
		}

		for _, withdraw := range withdraws {
			address := withdraw.Address
			found := false

			for _, acc := range baccounts {
				if acc.ID == withdraw.AccountID {
					address = acc.Address
					found = true
					break
				}
			}

			if !found {
				for _, acc := range aaccounts {
					if acc.ID == withdraw.AccountID {
						address = acc.Address
						break
					}
				}
			}

			_, err := cli.
				Withdraw.
				UpdateOneID(withdraw.ID).
				SetAddress(address).
				Save(_ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func Migrate(ctx context.Context) error {
	if err := db.Init(); err != nil {
		logger.Sugar().Errorw("Migrate", "error", err)
		return err
	}

	if err := migrateWithdrawAddress(ctx); err != nil {
		logger.Sugar().Errorw("Migrate", "error", err)
		return err
	}

	return nil
}
