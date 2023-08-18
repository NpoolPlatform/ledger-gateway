package profit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	statementhandler "github.com/NpoolPlatform/ledger-gateway/pkg/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	orderpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	"github.com/shopspring/decimal"
)

func (h *Handler) GetMiningRewards(ctx context.Context) ([]*npool.MiningReward, uint32, error) {
	handler := &statementhandler.Handler{
		Handler: h.Handler,
	}
	statements, total, err := handler.GetStatements(ctx)
	if err != nil {
		return nil, 0, err
	}

	ofs := int32(0)
	lim := int32(100)
	var orders []*ordermwpb.Order

	for {
		ords, _, err := ordermwcli.GetOrders(ctx, &ordermwpb.Conds{
			AppID: &commonpb.StringVal{
				Op:    cruder.EQ,
				Value: *h.AppID,
			},
			UserID: &commonpb.StringVal{
				Op:    cruder.EQ,
				Value: *h.UserID,
			},
		}, ofs, lim)
		if err != nil {
			return nil, 0, err
		}
		if len(ords) == 0 {
			break
		}

		orders = append(orders, ords...)

		ofs += lim
	}
	orderMap := map[string]*ordermwpb.Order{}
	for _, order := range orders {
		orderMap[order.ID] = order
	}

	var infos []*npool.MiningReward
	for _, info := range statements {
		type extra struct {
			GoodID  string
			OrderID string
		}

		e := extra{}
		err := json.Unmarshal([]byte(info.IOExtra), &e)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid io extra")
		}

		order, ok := orderMap[e.OrderID]
		if !ok {
			logger.Sugar().Warn("order not exist, id(%v)", e.OrderID)
			continue
		}

		switch order.OrderState {
		case orderpb.OrderState_Paid:
		case orderpb.OrderState_InService:
		case orderpb.OrderState_Expired:
		default:
			continue
		}

		rewardAmount, err := decimal.NewFromString(info.Amount)
		if err != nil {
			logger.Sugar().Warnw("GetMiningRewards", "amount cannot convert to number", info.Amount)
			continue
		}

		units, err := decimal.NewFromString(order.Units)
		if err != nil {
			logger.Sugar().Warnw("GetMiningRewards", "order units cannot convert to number", order.Units)
			continue
		}

		infos = append(infos, &npool.MiningReward{
			CoinTypeID:          info.CoinTypeID,
			CoinName:            info.CoinName,
			CoinLogo:            info.CoinLogo,
			CoinUnit:            info.CoinUnit,
			IOType:              info.IOType,
			IOSubType:           info.IOSubType,
			RewardAmount:        info.Amount,
			RewardAmountPerUnit: rewardAmount.Div(units).String(),
			Units:               order.Units,
			Extra:               info.IOExtra,
			GoodID:              e.GoodID,
			OrderID:             e.OrderID,
			CreatedAt:           info.CreatedAt,
		})
	}
	return infos, total, nil
}
