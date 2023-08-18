package transfer

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
	AppID            *string
	UserID           *string
	Account          *string
	AccountType      basetypes.SignMethod
	VerificationCode *string
	TargetUserID     *string
	CoinTypeID       *string
	Amount           *string
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

func WithVerificationCode(code *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if code == nil {
			if must {
				return fmt.Errorf("invalid code")
			}
			return nil
		}

		h.VerificationCode = code
		return nil
	}
}

func WithAccount(account *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if account == nil {
			if must {
				return fmt.Errorf("invalid account")
			}
			return nil
		}

		h.Account = account
		return nil
	}
}

func WithAccountType(accountType *basetypes.SignMethod, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if accountType == nil {
			if must {
				return fmt.Errorf("invalid account type")
			}
			return nil
		}
		switch *accountType {
		case basetypes.SignMethod_Email:
		case basetypes.SignMethod_Google:
		default:
			return fmt.Errorf("invalid account type %v", *accountType)
		}

		h.AccountType = *accountType
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
		_amount, err := decimal.NewFromString(*amount)
		if err != nil {
			return err
		}
		if _amount.Cmp(decimal.NewFromInt(0)) <= 0 {
			return fmt.Errorf("invalid amount %v", *amount)
		}
		h.Amount = amount
		return nil
	}
}
