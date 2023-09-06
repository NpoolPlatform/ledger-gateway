package profit

import (
	"context"

	types "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	statementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	"github.com/shopspring/decimal"
)

type rewardHandler struct {
	*baseHandler
	rewards []*npool.MiningReward
}

// Export In Frontend
func (h *rewardHandler) formalize() {
	for appGoodID, goodStatements := range h.statements {
		_, ok := h.appGoods[appGoodID]
		if !ok {
			continue
		}
		for coinTypeID, coinStatements := range goodStatements {
			coin, ok := h.appCoins[coinTypeID]
			if !ok {
				continue
			}
			for orderID, statements := range coinStatements {
				order, ok := h.orders[orderID]
				if !ok {
					continue
				}
				switch order.OrderState {
				case ordertypes.OrderState_OrderStatePaid:
				case ordertypes.OrderState_OrderStateInService:
				case ordertypes.OrderState_OrderStateExpired:
				default:
					continue
				}
				for _, val := range statements {
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
						GoodID:              appGoodID,
						OrderID:             orderID,
						CreatedAt:           val.CreatedAt,
					})
				}
			}
		}
	}
}

func (h *Handler) GetMiningRewards(ctx context.Context) ([]*npool.MiningReward, uint32, error) {
	handler := &rewardHandler{
		baseHandler: &baseHandler{
			Handler:    h,
			appCoins:   map[string]*appcoinmwpb.Coin{},
			appGoods:   map[string]*appgoodmwpb.Good{},
			orders:     map[string]*ordermwpb.Order{},
			statements: map[string]map[string]map[string][]*statementmwpb.Statement{},
			ioType:     types.IOType_Incoming,
			ioSubTypes: []types.IOSubType{types.IOSubType_MiningBenefit},
		},
	}
	if err := handler.getAppGoods(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getStatements(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()
	return handler.rewards, handler.total, nil
}
