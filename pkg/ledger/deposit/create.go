package deposit

import (
	"context"
	"fmt"
	"time"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	ledgerpb "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"

	appusermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/statement"
	statementpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
)

type createHandler struct {
	*Handler
	user *appusermwpb.User
}

func (h *createHandler) checkUser(ctx context.Context) error {
	if h.UserID == nil {
		return nil
	}
	user, err := usermwcli.GetUser(ctx, *h.AppID, *h.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("invalid user")
	}
	h.user = user
	return nil
}

func (h *Handler) CreateDeposit(ctx context.Context) (*npool.Statement, error) {
	handler := &createHandler{
		Handler: h,
	}
	if err := handler.checkUser(ctx); err != nil {
		return nil, err
	}
	coin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.TargetAppID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CoinTypeID},
	})
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid coin")
	}

	ioExtra := fmt.Sprintf(
		`{"AppID":"%v","UserID":"%v","TargetAppID":"%v","TargetUserID":"%v","CoinName":"%v","Amount":"%v","Date":"%v"}`,
		*h.AppID,
		*h.UserID,
		*h.TargetAppID,
		*h.TargetUserID,
		coin.Name,
		*h.Amount,
		time.Now(),
	)

	ioType := ledgerpb.IOType_Incoming
	ioSubtype := ledgerpb.IOSubType_Deposit
	info, err := ledgermwcli.CreateStatement(ctx, &statementpb.StatementReq{
		AppID:      h.TargetAppID,
		UserID:     h.TargetUserID,
		CoinTypeID: h.CoinTypeID,
		IOType:     &ioType,
		IOSubType:  &ioSubtype,
		Amount:     h.Amount,
		IOExtra:    &ioExtra,
	})
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}

	return &npool.Statement{
		ID:           info.ID,
		EntID:        info.EntID,
		UserID:       info.UserID,
		EmailAddress: handler.user.EmailAddress,
		CoinTypeID:   *h.CoinTypeID,
		CoinName:     coin.Name,
		DisplayNames: coin.DisplayNames,
		CoinLogo:     coin.Logo,
		CoinUnit:     coin.Unit,
		IOType:       info.IOType,
		IOSubType:    info.IOSubType,
		Amount:       info.Amount,
		IOExtra:      info.IOExtra,
		CreatedAt:    info.CreatedAt,
	}, nil
}
