package statement

import (
	"context"

	appcoincli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	statementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	commonpb "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	"github.com/NpoolPlatform/message/npool/ledger/mw/v2/statement"
	"github.com/google/uuid"
)

func (h *Handler) setConds() *statement.Conds {
	conds := &statement.Conds{}
	if h.AppID != nil {
		conds.AppID = &commonpb.StringVal{Op: cruder.EQ, Value: *h.AppID}
	}
	if h.UserID != nil {
		conds.UserID = &commonpb.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}
	if h.StartAt != nil {
		conds.StartAt = &commonpb.Uint32Val{Op: cruder.EQ, Value: *h.StartAt}
	}
	if h.EndAt != nil {
		conds.EndAt = &commonpb.Uint32Val{Op: cruder.EQ, Value: *h.EndAt}
	}
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
	coins, _, err := appcoincli.GetCoins(ctx, &appcoinpb.Conds{
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
	coinMap := map[string]*appcoinpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.CoinTypeID] = coin
	}

	infos := []*npool.Statement{}
	for _, info := range statements {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			logger.Sugar().Warnw("GetStatements", "app coin not exist", "appid", coin.AppID, "cointypeid", coin.CoinTypeID)
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
		})
	}

	return infos, total, nil
}
