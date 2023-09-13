package profit

import (
	"context"
	"encoding/json"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	constant "github.com/NpoolPlatform/ledger-gateway/pkg/const"
	statementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	ledgertypes "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	statementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	"github.com/shopspring/decimal"
)

type goodProfitHandler struct {
	*Handler
	infos       []*npool.GoodProfit
	statements  map[string]map[string]map[string][]*statementmwpb.Statement // AppGoodID -> CoinTypeID -> OrderID
	appCoins    map[string]*appcoinmwpb.Coin
	appGoods    map[string]*appgoodmwpb.Good
	orders      map[string]*ordermwpb.Order
	coinTypeIDs []string
}

//nolint
func (h *goodProfitHandler) calculateOrderProfit(orderID string, statements []*statementmwpb.Statement) (decimal.Decimal, decimal.Decimal) {
	incoming := decimal.NewFromInt(0)
	units := decimal.NewFromInt(0)

	for _, val := range statements {
		order, ok := h.orders[orderID]
		if !ok {
			continue
		}
		incoming = incoming.Add(decimal.RequireFromString(val.Amount))
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
		AppGoodID:             appGoodID,
		GoodName:              good.GoodName,
		GoodUnit:              good.Unit,
		GoodServicePeriodDays: uint32(good.DurationDays),
		Units:                 units.String(),
		Incoming:              amount.String(),
	})
}

func (h *goodProfitHandler) formalize() {
	for _, good := range h.appGoods {
		goodStatements, ok := h.statements[good.ID]
		if !ok {
			h.formalizeProfit(good.ID, good.CoinTypeID, decimal.NewFromInt(0), decimal.NewFromInt(0))
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
			h.formalizeProfit(good.ID, coinTypeID, coinProfitAmount, coinOrderUnits)
		}
	}
}

//nolint
func (h *goodProfitHandler) getOrders(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit
	for {
		orders, _, err := ordermwcli.GetOrders(ctx, &ordermwpb.Conds{
			AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			OrderStates: &basetypes.Uint32SliceVal{Op: cruder.NIN, Value: []uint32{
				uint32(ordertypes.OrderState_OrderStateCreated),
				uint32(ordertypes.OrderState_OrderStateWaitPayment),
				uint32(ordertypes.OrderState_OrderStatePaymentTimeout),
				uint32(ordertypes.OrderState_OrderStatePreCancel),
				uint32(ordertypes.OrderState_OrderStateRestoreCanceledStock),
				uint32(ordertypes.OrderState_OrderStateCancelAchievement),
				uint32(ordertypes.OrderState_OrderStateDeductLockedCommission),
				uint32(ordertypes.OrderState_OrderStateReturnCanceledBalance),
				uint32(ordertypes.OrderState_OrderStateCanceledTransferBookKeeping),
				uint32(ordertypes.OrderState_OrderStateCancelUnlockPaymentAccount),
				uint32(ordertypes.OrderState_OrderStateUpdateCanceledChilds),
				uint32(ordertypes.OrderState_OrderStateCanceled),
			}},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			break
		}

		for _, order := range orders {
			h.orders[order.ID] = order
		}
		offset += limit
	}
	return nil
}

func (h *Handler) getAppGoods(ctx context.Context) ([]*appgoodmwpb.Good, uint32, error) {
	goods, total, err := appgoodmwcli.GetGoods(ctx, &appgoodmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}
	return goods, total, nil
}

func (h *goodProfitHandler) getAppCoins(ctx context.Context) error {
	for _, val := range h.appGoods {
		h.coinTypeIDs = append(h.coinTypeIDs, val.CoinTypeID)
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.coinTypeIDs},
	}, 0, int32(len(h.coinTypeIDs)))
	if err != nil {
		return err
	}
	for _, coin := range coins {
		h.appCoins[coin.CoinTypeID] = coin
	}
	return nil
}

func (h *goodProfitHandler) getStatements(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit
	for {
		statements, _, err := statementcli.GetStatements(ctx, &statementmwpb.Conds{
			AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			IOType:      &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ledgertypes.IOType_Incoming)},
			IOSubType:   &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ledgertypes.IOSubType_MiningBenefit)},
			StartAt:     &basetypes.Uint32Val{Op: cruder.EQ, Value: h.StartAt},
			EndAt:       &basetypes.Uint32Val{Op: cruder.EQ, Value: h.EndAt},
			CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.coinTypeIDs},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(statements) == 0 {
			break
		}
		for _, statement := range statements {
			e := struct {
				OrderID   string
				AppGoodID string
			}{}
			if err := json.Unmarshal([]byte(statement.IOExtra), &e); err != nil {
				logger.Sugar().Errorw(
					"getStatements",
					"IOExtra", statement.IOExtra,
					"Error", err,
				)
				continue
			}
			order, ok := h.orders[e.OrderID]
			if !ok {
				logger.Sugar().Errorw(
					"getStatements",
					"OrderID", e.OrderID,
					"Error", "Invalid order",
				)
				continue
			}
			if order.AppGoodID != e.AppGoodID {
				logger.Sugar().Errorw(
					"getStatements",
					"OrderAppGoodID", order.AppGoodID,
					"OrderGoodID", order.GoodID,
					"StatementAppGoodID", e.AppGoodID,
					"Error", "Invalid statement",
				)
				continue
			}
			goodStatements, ok := h.statements[order.AppGoodID]
			if !ok {
				goodStatements = map[string]map[string][]*statementmwpb.Statement{}
			}
			coinStatements, ok := goodStatements[order.CoinTypeID]
			if !ok {
				coinStatements = map[string][]*statementmwpb.Statement{}
			}
			orderStatements, ok := coinStatements[order.ID]
			if !ok {
				orderStatements = []*statementmwpb.Statement{}
			}
			orderStatements = append(orderStatements, statement)
			coinStatements[order.ID] = orderStatements
			goodStatements[order.CoinTypeID] = coinStatements
			h.statements[order.AppGoodID] = goodStatements
		}
		offset += limit
	}
	return nil
}

func (h *Handler) GetGoodProfits(ctx context.Context) ([]*npool.GoodProfit, uint32, error) {
	goods, total, err := h.getAppGoods(ctx)
	if err != nil {
		return nil, 0, err
	}
	if len(goods) == 0 {
		return nil, total, nil
	}
	goodMap := map[string]*appgoodmwpb.Good{}
	for _, good := range goods {
		goodMap[good.ID] = good
	}

	handler := &goodProfitHandler{
		Handler:    h,
		appCoins:   map[string]*appcoinmwpb.Coin{},
		orders:     map[string]*ordermwpb.Order{},
		statements: map[string]map[string]map[string][]*statementmwpb.Statement{},
		appGoods:   goodMap,
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getStatements(ctx); err != nil {
		return nil, 0, err
	}
	handler.formalize()
	return handler.infos, total, nil
}
