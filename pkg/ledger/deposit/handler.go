package deposit

import (
	"context"
	"fmt"

	appcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	appusercli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	"github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handler struct {
	ID           *string
	AppID        *string
	UserID       *string
	CoinTypeID   *string
	TargetAppID  *string
	TargetUserID *string
	Amount       *string
}

func NewHandler(ctx context.Context, options ...func(context.Context, *Handler) error) (*Handler, error) {
	handler := &Handler{}
	for _, opt := range options {
		if err := opt(ctx, handler); err != nil {
			return nil, err
		}
	}
	return handler, nil
}

func WithAppID(appID *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if appID == nil {
			return fmt.Errorf("invalid app id")
		}
		_, err := uuid.Parse(*appID)
		if err != nil {
			return err
		}
		exist, err := appcli.ExistApp(ctx, *appID)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("invalid app")
		}

		h.AppID = appID
		return nil
	}
}

func WithUserID(appID, userID *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if userID == nil {
			if must {
				return fmt.Errorf("invalid user id")
			}
			return nil
		}
		_, err := uuid.Parse(*userID)
		if err != nil {
			return err
		}

		exist, err := appusercli.ExistUser(ctx, *appID, *userID)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("invalid app id or user id")
		}
		h.UserID = userID
		return nil
	}
}

func WithTargetAppID(appID *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if appID == nil {
			if must {
				return fmt.Errorf("invalid target app id")
			}
			return nil
		}
		_, err := uuid.Parse(*appID)
		if err != nil {
			return err
		}
		exist, err := appcli.ExistApp(ctx, *appID)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("invalid app")
		}

		h.TargetAppID = appID
		return nil
	}
}

func WithTargetUserID(appID, userID *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if userID == nil {
			if must {
				return fmt.Errorf("invalid target user id")
			}
			return nil
		}
		_, err := uuid.Parse(*userID)
		if err != nil {
			return err
		}

		exist, err := appusercli.ExistUser(ctx, *appID, *userID)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("invalid app id or user id")
		}
		h.TargetUserID = userID
		return nil
	}
}

func WithCoinTypeID(appID, coinTypeID *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if coinTypeID == nil {
			if must {
				return fmt.Errorf("invalid coin type id")
			}
			return nil
		}
		_, err := uuid.Parse(*coinTypeID)
		if err != nil {
			return err
		}

		exist, err := appcoinmwcli.ExistCoinConds(ctx, &coin.Conds{
			AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *appID},
			CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *coinTypeID},
		})
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("coin not exist %v", *coinTypeID)
		}
		h.CoinTypeID = coinTypeID
		return nil
	}
}

func WithAmount(amount *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if amount == nil {
			if must {
				return fmt.Errorf("invalid amount")
			}
			return nil
		}
		_, err := decimal.NewFromString(*amount)
		if err != nil {
			return err
		}

		h.Amount = amount
		return nil
	}
}
