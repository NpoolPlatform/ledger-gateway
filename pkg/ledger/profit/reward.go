package profit

import (
	"context"
	"encoding/json"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	types "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	orderpb "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	"github.com/shopspring/decimal"
)

type rewardHandler struct {
	*baseHandler
	rewards []*npool.MiningReward
}

// Export In Frontend
func (h *rewardHandler) formalize() {
	for appGoodID, good := range h.appGoods {
		goodStatements, ok := h.statements[appGoodID]
		if !ok {
			continue
		}

		for coinTypeID, coinStatement := range goodStatements {
			
		}

	}
	for _, val := range h.statements {
		type extra struct {
			GoodID  string
			OrderID string
		}

		e := extra{}
		err := json.Unmarshal([]byte(val.IOExtra), &e)
		if err != nil {
			continue
		}

		order, ok := h.orders[e.OrderID]
		if !ok {
			logger.Sugar().Errorf("invalid order id %v", e.OrderID)
			continue
		}

		coin, ok := h.appCoins[val.CoinTypeID]
		if !ok {
			logger.Sugar().Errorf("invalid coin type id %v", val.CoinTypeID)
			continue
		}

		switch order.OrderState {
		case orderpb.OrderState_OrderStatePaid:
		case orderpb.OrderState_OrderStateInService:
		case orderpb.OrderState_OrderStateExpired:
		default:
			continue
		}

		rewardAmount, err := decimal.NewFromString(val.Amount)
		if err != nil {
			continue
		}
		units, err := decimal.NewFromString(order.Units)
		if err != nil {
			continue
		}

		h.rewards = append(h.rewards, &npool.MiningReward{
			CoinTypeID:          val.CoinTypeID,
			CoinName:            coin.CoinName,
			CoinLogo:            coin.Logo,
			CoinUnit:            coin.Unit,
			IOType:              val.IOType,
			IOSubType:           val.IOSubType,
			RewardAmount:        val.Amount,
			RewardAmountPerUnit: rewardAmount.Div(units).String(),
			Units:               order.Units,
			Extra:               val.IOExtra,
			GoodID:              e.GoodID,
			OrderID:             e.OrderID,
			CreatedAt:           val.CreatedAt,
		})
	}
}

func (h *Handler) GetMiningRewards(ctx context.Context) ([]*npool.MiningReward, uint32, error) {
	handler := &rewardHandler{
		baseHandler: &baseHandler{
			Handler:    h,
			appCoins:   map[string]*appcoinmwpb.Coin{},
			orders:     map[string]*ordermwpb.Order{},
			ioType:     types.IOType_Incoming,
			ioSubTypes: []types.IOSubType{types.IOSubType_MiningBenefit},
		},
	}
	if err := handler.getOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getStatements(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()
	return handler.rewards, handler.total, nil
}
