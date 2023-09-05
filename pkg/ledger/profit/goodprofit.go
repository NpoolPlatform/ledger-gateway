package profit

import (
	"context"
	"encoding/json"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	types "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	orderpb "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	goodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	"github.com/shopspring/decimal"
)

type goodProfitHandler struct {
	*BaseHandler
	profits []*npool.GoodProfit
}

//nolint
func (h *goodProfitHandler) formalize() {
	type extra struct {
		BenefitDate string
		OrderID     string
	}

	infos := map[string]*npool.GoodProfit{}

	profitOrderMap := map[string]struct{}{}
	for _, info := range h.statements {
		e := extra{}
		err := json.Unmarshal([]byte(info.IOExtra), &e)
		if err != nil {
			logger.Sugar().Warnw("invalid io extra", "ioextra", info.IOExtra)
			continue
		}

		order, ok := h.orders[e.OrderID]
		if !ok {
            logger.Sugar().Errorf("invalid order %v", e.OrderID)
			continue
		}

		switch order.OrderState {
		case orderpb.OrderState_OrderStatePaid:
		case orderpb.OrderState_OrderStateInService:
		case orderpb.OrderState_OrderStateExpired:
		default:
			continue
		}

		good, ok := h.goods[order.GoodID]
		if !ok {
            logger.Sugar().Errorf("invalid good %v", order.GoodID)
			continue
		}

		coin, ok := h.appCoins[good.CoinTypeID]
		if !ok {
            logger.Sugar().Errorf("invalid coin type id %v", good.CoinTypeID)
			continue
		}

		gp, ok := infos[order.GoodID]
		if !ok {
			gp = &npool.GoodProfit{
				CoinTypeID:            good.CoinTypeID,
				CoinName:              coin.Name,
				DisplayNames:          coin.DisplayNames,
				CoinLogo:              coin.Logo,
				CoinUnit:              coin.Unit,
				GoodID:                order.GoodID,
				GoodName:              good.Title,
				GoodUnit:              good.Unit,
				GoodServicePeriodDays: uint32(good.DurationDays),
				Units:                 decimal.NewFromInt(0).String(),
				Incoming:              decimal.NewFromInt(0).String(),
			}
		}

		if info.IOSubType == types.IOSubType_MiningBenefit {
			gp.Incoming = decimal.RequireFromString(gp.Incoming).
				Add(decimal.RequireFromString(info.Amount)).
				String()
		}

		if _, ok := profitOrderMap[order.ID]; !ok {
			gp.Units = decimal.RequireFromString(gp.Units).
				Add(decimal.RequireFromString(order.Units)).
				String()
		}

		profitOrderMap[order.ID] = struct{}{}
		infos[order.GoodID] = gp
	}

	for _, order := range h.orders {
		if _, ok := profitOrderMap[order.ID]; ok {
			continue
		}

		switch order.OrderState {
		case orderpb.OrderState_OrderStatePaid:
		case orderpb.OrderState_OrderStateInService:
		case orderpb.OrderState_OrderStateExpired:
		default:
			continue
		}

		good, ok := h.goods[order.GoodID]
		if !ok {
			logger.Sugar().Warn("good %v not exist", order.GoodID)
			continue
		}

		coin, ok := h.appCoins[good.CoinTypeID]
		if !ok {
			logger.Sugar().Warn("coin not exist %v", good.CoinTypeID)
			continue
		}

		gp, ok := infos[order.GoodID]
		if !ok {
			gp = &npool.GoodProfit{
				CoinTypeID:            good.CoinTypeID,
				CoinName:              coin.Name,
				DisplayNames:          coin.DisplayNames,
				CoinLogo:              coin.Logo,
				CoinUnit:              coin.Unit,
				GoodID:                order.GoodID,
				GoodName:              good.Title,
				GoodUnit:              good.Unit,
				GoodServicePeriodDays: uint32(good.DurationDays),
				Units:                 decimal.NewFromInt(0).String(),
				Incoming:              decimal.NewFromInt(0).String(),
			}
		}

		gp.Units = decimal.
			RequireFromString(gp.Units).
			Add(decimal.RequireFromString(order.Units)).
			String()
		infos[order.GoodID] = gp
	}

	for _, info := range infos {
		h.profits = append(h.profits, info)
	}
}

func (h *Handler) GetGoodProfits(ctx context.Context) ([]*npool.GoodProfit, uint32, error) {
	handler := &goodProfitHandler{
		BaseHandler: &BaseHandler{
			Handler:  h,
			appCoins: map[string]*appcoinmwpb.Coin{},
			orders:   map[string]*ordermwpb.Order{},
			goods:    map[string]*goodmwpb.Good{},
		},
	}
	if err := handler.getStatements(ctx, types.IOType_Incoming, []types.IOSubType{types.IOSubType_MiningBenefit, types.IOSubType_Payment}); err != nil {
		return nil, 0, err
	}
	if len(handler.statements) == 0 {
		return nil, 0, nil
	}

	if err := handler.getOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getGoods(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()
	return handler.profits, handler.total, nil
}
