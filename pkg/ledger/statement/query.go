package statement

import (
	"context"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	statementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	appusermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/statement"
	statementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
)

type queryHandler struct {
	*Handler
	statements []*statementmwpb.Statement
	appcoin    map[string]*appcoinmwpb.Coin
	appuser    map[string]*appusermwpb.User
	infos      []*npool.Statement
}

func (h *queryHandler) getAppCoins(ctx context.Context) error {
	coinTypeIDs := []string{}
	for _, val := range h.statements {
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return err
	}

	for _, coin := range coins {
		h.appcoin[coin.CoinTypeID] = coin
	}
	return nil
}

func (h *queryHandler) getAppUsers(ctx context.Context) error {
	userIDs := []string{}
	for _, info := range h.statements {
		userIDs = append(userIDs, info.UserID)
	}

	users, _, err := usermwcli.GetUsers(ctx, &appusermwpb.Conds{
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: userIDs},
	}, 0, int32(len(userIDs)))
	if err != nil {
		return err
	}

	for _, user := range users {
		h.appuser[user.EntID] = user
	}
	return nil
}

func (h *queryHandler) formalize() {
	for _, statement := range h.statements {
		coin, ok := h.appcoin[statement.CoinTypeID]
		if !ok {
			continue
		}
		user, ok := h.appuser[statement.UserID]
		if !ok {
			continue
		}

		h.infos = append(h.infos, &npool.Statement{
			ID:           statement.ID,
			AppID:        statement.AppID,
			CoinTypeID:   coin.CoinTypeID,
			CoinName:     coin.CoinName,
			DisplayNames: coin.DisplayNames,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			IOType:       statement.IOType,
			IOSubType:    statement.IOSubType,
			IOExtra:      statement.IOExtra,
			Amount:       statement.Amount,
			CreatedAt:    statement.CreatedAt,
			UserID:       user.EntID,
			PhoneNO:      user.PhoneNO,
			EmailAddress: user.EmailAddress,
		})
	}
}

func (h *Handler) GetStatements(ctx context.Context) ([]*npool.Statement, uint32, error) {
	conds := &statementmwpb.Conds{}
	if h.AppID != nil {
		conds.AppID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID}
	}
	if h.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}
	conds.StartAt = &basetypes.Uint32Val{Op: cruder.EQ, Value: h.StartAt}
	if h.EndAt != 0 {
		conds.EndAt = &basetypes.Uint32Val{Op: cruder.EQ, Value: h.EndAt}
	}
	statements, total, err := statementcli.GetStatements(ctx, conds, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}
	if len(statements) == 0 {
		return nil, total, nil
	}

	handler := &queryHandler{
		Handler:    h,
		statements: statements,
		appcoin:    map[string]*appcoinmwpb.Coin{},
		appuser:    map[string]*appusermwpb.User{},
	}

	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppUsers(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()
	return handler.infos, total, nil
}
