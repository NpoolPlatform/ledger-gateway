package migrator

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	"github.com/NpoolPlatform/ledger-gateway/pkg/servicename"
	"github.com/NpoolPlatform/ledger-middleware/pkg/db"
	"github.com/NpoolPlatform/ledger-middleware/pkg/db/ent"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

const keyServiceID = "ledger-gateway"

func lockKey() string {
	serviceID := config.GetStringValueWithNameSpace(servicename.ServiceName, keyServiceID)
	return fmt.Sprintf("%v:%v", basetypes.Prefix_PrefixMigrate, serviceID)
}

func Migrate(ctx context.Context) error {
	var err error
	if err := db.Init(); err != nil {
		return err
	}

	logger.Sugar().Infow("Migrate", "Start", "...")
	defer func() {
		_ = redis2.Unlock(lockKey())
		logger.Sugar().Infow("Migrate", "Done", "...", "error", err)
	}()
	err = redis2.TryLock(lockKey(), 0)
	if err != nil {
		return err
	}

	err = db.WithTx(ctx, func(ctx context.Context, tx *ent.Tx) error {
		return nil
	})
	return err
}
