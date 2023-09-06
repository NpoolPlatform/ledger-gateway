package profit

import (
	"context"

	types "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	orderpb "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	statementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	"github.com/shopspring/decimal"
)

type goodProfitHandler struct {
	*baseHandler
	infos []*npool.GoodProfit
}

//nolint
func (h *goodProfitHandler) calculateOrderProfit(orderID string, statements []*statementmwpb.Statement) (decimal.Decimal, decimal.Decimal) {
	incoming := decimal.NewFromInt(0)
	units := decimal.NewFromInt(0)

	for _, val := range statements {
		order, ok := h.orders[orderID]
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
<<<<<<< HEAD

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
=======
		incoming = incoming.Add(decimal.RequireFromString(val.Amount))
		units = units.Add(decimal.RequireFromString(order.Units))
>>>>>>> 00047b5422539acf4a20f729f4177a41a306f60f
	}
	return incoming, units
}

func (h *goodProfitHandler) formalizeProfit(appGoodID, coinTypeID string, amount, units decimal.Decimal) {
	good, ok := h.appGoods[appGoodID]
	if !ok {
		return
	}
	coin, ok := h.appCoins[coinTypeID]
	if !ok {
		return
	}

	h.infos = append(h.infos, &npool.GoodProfit{
		CoinTypeID:            good.CoinTypeID,
		CoinName:              coin.Name,
		DisplayNames:          coin.DisplayNames,
		CoinLogo:              coin.Logo,
		CoinUnit:              coin.Unit,
		GoodID:                appGoodID,
		GoodName:              good.GoodName,
		GoodUnit:              good.Unit,
		GoodServicePeriodDays: uint32(good.DurationDays),
		Units:                 units.String(),
		Incoming:              amount.String(),
	})
}

func (h *goodProfitHandler) formalize() {
	for appGoodID, good := range h.appGoods {
		goodStatements, ok := h.statements[appGoodID]
		if !ok {
			h.formalizeProfit(appGoodID, good.CoinTypeID, decimal.NewFromInt(0), decimal.NewFromInt(0))
			continue
		}

		for coinTypeID, coinStatements := range goodStatements {
			_, ok := h.appCoins[coinTypeID]
			if !ok {
				continue
			}

			coinProfitAmount := decimal.NewFromInt(0)
			coinOrderUnits := decimal.NewFromInt(0)
			for orderID, statements := range coinStatements {
				amount, units := h.calculateOrderProfit(orderID, statements)
				coinProfitAmount = coinProfitAmount.Add(amount)
				coinOrderUnits = coinOrderUnits.Add(units)
			}
			h.formalizeProfit(appGoodID, coinTypeID, coinProfitAmount, coinOrderUnits)
		}
	}
}

func (h *Handler) GetGoodProfits(ctx context.Context) ([]*npool.GoodProfit, uint32, error) {
	handler := &goodProfitHandler{
		baseHandler: &baseHandler{
			Handler:    h,
			appCoins:   map[string]*appcoinmwpb.Coin{},
			orders:     map[string]*ordermwpb.Order{},
			appGoods:   map[string]*appgoodmwpb.Good{},
			ioType:     types.IOType_Incoming,
			ioSubTypes: []types.IOSubType{types.IOSubType_MiningBenefit, types.IOSubType_Payment},
		},
	}
	if err := handler.getOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppGoods(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getStatements(ctx); err != nil {
		return nil, 0, err
	}
	handler.formalize()
	return handler.infos, handler.total, nil
}
