package statement

import (
	"context"
	"fmt"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	statementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	"github.com/NpoolPlatform/message/npool/ledger/mw/v2/statement"
	"github.com/google/uuid"
)

func (h *Handler) setConds() *statement.Conds {
	conds := &statement.Conds{}
	if h.AppID != nil {
		conds.AppID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID}
	}
	if h.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}
	conds.StartAt = &basetypes.Uint32Val{Op: cruder.GT, Value: h.StartAt}
	conds.EndAt = &basetypes.Uint32Val{Op: cruder.LT, Value: h.EndAt}
	return conds
}

func (h *Handler) GetStatements(ctx context.Context) ([]*npool.Statement, uint32, error) {
	statements, total, err := statementcli.GetStatements(ctx, h.setConds(), h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}
	if len(statements) == 0 {
		return nil, 0, nil
	}

	coinTypeIDs := []string{}
	for _, val := range statements {
		if _, err := uuid.Parse(val.CoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
		CoinTypeIDs: &basetypes.StringSliceVal{
			Op:    cruder.IN,
			Value: coinTypeIDs,
		},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return nil, 0, err
	}
	coinMap := map[string]*appcoinmwpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.CoinTypeID] = coin
	}

	userIDs := []string{}
	for _, info := range statements {
		userIDs = append(userIDs, info.UserID)
	}

	users, _, err := usermwcli.GetUsers(ctx, &usermwpb.Conds{
		IDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: userIDs},
	}, 0, int32(len(userIDs)))
	if err != nil {
		return nil, 0, fmt.Errorf("fail get users: %v", err)
	}

	userMap := map[string]*usermwpb.User{}
	for _, user := range users {
		userMap[user.ID] = user
	}

	infos := []*npool.Statement{}
	for _, info := range statements {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			logger.Sugar().Warnw("GetStatements", "app coin not exist", "appid", coin.AppID, "cointypeid", coin.CoinTypeID)
			continue
		}
		user, ok := userMap[info.UserID]
		if !ok {
			continue
		}

		infos = append(infos, &npool.Statement{
			CoinTypeID:   info.CoinTypeID,
			CoinName:     coin.Name,
			DisplayNames: coin.DisplayNames,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			IOType:       info.IOType,
			IOSubType:    info.IOSubType,
			Amount:       info.Amount,
			IOExtra:      info.IOExtra,
			CreatedAt:    info.CreatedAt,
			UserID:       info.UserID,
			PhoneNO:      user.PhoneNO,
			EmailAddress: user.EmailAddress,
		})
	}

	return infos, total, nil
}
