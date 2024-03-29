package withdraw

import (
	"context"
	"fmt"

	appmwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	constant "github.com/NpoolPlatform/ledger-gateway/pkg/const"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handler struct {
	ID                    *uint32
	EntID                 *string
	AppID                 *string
	UserID                *string
	VerificationCode      *string
	Account               *string
	AccountType           *basetypes.SignMethod
	CoinTypeID            *string
	AccountID             *string
	Amount                *string
	Offset                int32
	Limit                 int32
	RequestTimeoutSeconds *int64
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

func WithID(id *uint32, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid id")
			}
			return nil
		}
		h.ID = id
		return nil
	}
}

func WithEntID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid entid")
			}
			return nil
		}
		if _, err := uuid.Parse(*id); err != nil {
			return err
		}
		h.EntID = id
		return nil
	}
}

func WithAppID(appID *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if appID == nil {
			if must {
				return fmt.Errorf("invalid app id")
			}
			return nil
		}
		if _, err := uuid.Parse(*appID); err != nil {
			return err
		}
		exist, err := appmwcli.ExistApp(ctx, *appID)
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

func WithUserID(userID *string, must bool) func(context.Context, *Handler) error {
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
		h.UserID = userID
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
		flag := false
		for _type := range basetypes.SignMethod_value {
			if accountType.String() == _type && accountType.String() != basetypes.SignMethod_DefaultSignMethod.String() {
				flag = true
			}
		}
		if !flag {
			return fmt.Errorf("invalid account type %v", accountType.String())
		}

		switch *accountType {
		case basetypes.SignMethod_Email:
		case basetypes.SignMethod_Google:
		case basetypes.SignMethod_Mobile:
		default:
			return fmt.Errorf("invalid account type %v", *accountType)
		}

		h.AccountType = accountType
		return nil
	}
}

func WithAccountID(accountID *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if accountID == nil {
			if must {
				return fmt.Errorf("invalid account id")
			}
			return nil
		}
		if _, err := uuid.Parse(*accountID); err != nil {
			return err
		}
		h.AccountID = accountID
		return nil
	}
}

func WithCoinTypeID(coinTypeID *string, must bool) func(context.Context, *Handler) error {
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
		if _, err := decimal.NewFromString(*amount); err != nil {
			return err
		}
		h.Amount = amount
		return nil
	}
}

func WithOffset(offset int32) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		h.Offset = offset
		return nil
	}
}

func WithLimit(limit int32) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if limit == 0 {
			limit = constant.DefaultRowLimit
		}
		h.Limit = limit
		return nil
	}
}
