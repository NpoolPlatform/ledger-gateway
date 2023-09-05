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
	infos []*npool.GoodProfit
}

func (h *goodProfitHandler) calculateOrderProfit(orderID string, statements []*statementmwpb.Statement) (decimal.Decimal, decimal.Decimal) {
	incoming := decimal.NewFromInt(0)
	units := decimal.NewFromInt(0)

	for _, statement := range statements {
		order, ok := h.orders[orderID]
		if !ok {
			continue
		}
		switch order.OrderState {
		case orderpb.OrderState_OrderStatePaid:
		case orderpb.OrderState_OrderStateInService:
		case orderpb.OrderState_OrderStateExpired:
		default:
			continue
		}
		incoming = incoming.Add(decimal.RequireFromString(info.Amount))
		units = units.Add(decimal.RequireFromString(order.Units))
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
		GoodName:              good.Title,
		GoodUnit:              good.Unit,
		GoodServicePeriodDays: uint32(good.DurationDays),
		Units:                 uints.String(),
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
			coin, ok := h.appCoins[coinTypeID]
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
		BaseHandler: &BaseHandler{
			Handler:    h,
			appCoins:   map[string]*appcoinmwpb.Coin{},
			orders:     map[string]*ordermwpb.Order{},
			goods:      map[string]*goodmwpb.Good{},
			ioType:     types.IOType_Incoming,
			ioSubTypes: []types.IOSubType{types.IOSubType_MiningBenefit, types.IOSubType_Payment},
		},
	}
	if err := handler.getStatements(ctx); err != nil {
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
