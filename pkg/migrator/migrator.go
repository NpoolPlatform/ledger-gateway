package migrator

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	constant1 "github.com/NpoolPlatform/ledger-gateway/pkg/const"
	"github.com/NpoolPlatform/ledger-gateway/pkg/servicename"
	"github.com/NpoolPlatform/ledger-middleware/pkg/db"
	"github.com/NpoolPlatform/ledger-middleware/pkg/db/ent"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	reviewtypes "github.com/NpoolPlatform/message/npool/basetypes/review/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	reviewmwpb "github.com/NpoolPlatform/message/npool/review/mw/v2/review"
	reviewmwcli "github.com/NpoolPlatform/review-middleware/pkg/client/review"
	"github.com/google/uuid"
)

const keyServiceID = "ledger-gateway"

func lockKey() string {
	serviceID := config.GetStringValueWithNameSpace(servicename.ServiceName, keyServiceID)
	return fmt.Sprintf("%v:%v", basetypes.Prefix_PrefixMigrate, serviceID)
}

func migrateReviewID(ctx context.Context, tx *ent.Tx) error {
	r, err := tx.QueryContext(ctx, "select id from withdraws where review_id is null")
	if err != nil {
		return err
	}

	type w struct {
		ID uuid.UUID
	}
	withdrawIDs := []uuid.UUID{}
	for r.Next() {
		_w := &w{}
		if err := r.Scan(&_w.ID); err != nil {
			return err
		}
		withdrawIDs = append(withdrawIDs, _w.ID)
	}
	if len(withdrawIDs) == 0 {
		return nil
	}

	offset := int32(0)
	limit := constant1.DefaultRowLimit
	reviews := map[string]*reviewmwpb.Review{}
	for {
		infos, _, err := reviewmwcli.GetReviews(ctx, &reviewmwpb.Conds{
			ObjectType: &basetypes.Int32Val{Op: cruder.EQ, Value: int32(reviewtypes.ReviewObjectType_ObjectWithdrawal)},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(infos) == 0 {
			break
		}

		for _, val := range infos {
			reviews[val.ObjectID] = val
		}
		offset += limit
	}

	for _, withdrawID := range withdrawIDs {
		review, ok := reviews[withdrawID.String()]
		if !ok {
			continue
		}
		reviewID := uuid.MustParse(review.ID)

		if _, err := tx.
			Withdraw.
			UpdateOneID(withdrawID).
			SetReviewID(reviewID).
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func Migrate(ctx context.Context) error {
	var err error

	if err := db.Init(); err != nil {
		return err
	}
	logger.Sugar().Infow("Migrate ReviewID", "Start", "...")
	defer func() {
		_ = redis2.Unlock(lockKey())
		logger.Sugar().Infow("Migrate ReviewID", "Done", "...", "error", err)
	}()

	err = redis2.TryLock(lockKey(), 0)
	if err != nil {
		return err
	}
	return db.WithTx(ctx, func(ctx context.Context, tx *ent.Tx) error {
		if err := migrateReviewID(ctx, tx); err != nil {
			return err
		}
		return nil
	})
}
