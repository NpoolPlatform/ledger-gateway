package profit

import (
	"context"
	"encoding/json"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	apppowerrentalmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/powerrental"
	goodcoinmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good/coin"
	constant "github.com/NpoolPlatform/ledger-gateway/pkg/const"
	statementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	ledgertypes "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	apppowerrentalmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/powerrental"
	goodcoinmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	statementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	powerrentalordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/powerrental"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	powerrentalordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/powerrental"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type goodProfitHandler struct {
	*Handler
	infos             []*npool.GoodProfit
	statements        map[string][]*statementmwpb.Statement
	appCoins          map[string]*appcoinmwpb.Coin
	appGoods          map[string]*appgoodmwpb.Good
	appPowerRentals   map[string]*apppowerrentalmwpb.PowerRental
	goodCoins         map[string][]*goodcoinmwpb.GoodCoin
	orders            map[string]*ordermwpb.Order
	powerRentalOrders []*powerrentalordermwpb.PowerRentalOrder
	coinTypeIDs       []string
	total             uint32
}

func (h *goodProfitHandler) formalizeProfit(appGoodID, coinTypeID string, goodMainCoin bool, amount decimal.Decimal) {
	good, ok := h.appGoods[appGoodID]
	if !ok {
		return
	}
	coin, ok := h.appCoins[coinTypeID]
	if !ok {
		return
	}

	units := decimal.NewFromInt(0)
	for _, powerRentalOrder := range h.powerRentalOrders {
		if powerRentalOrder.AppGoodID == appGoodID {
			_units, err := decimal.NewFromString(powerRentalOrder.Units)
			if err != nil {
				return
			}
			units = units.Add(_units)
		}
	}

	info := &npool.GoodProfit{
		AppID:        *h.AppID,
		UserID:       *h.UserID,
		AppGoodID:    appGoodID,
		AppGoodName:  good.AppGoodName,
		GoodType:     good.GoodType,
		CoinTypeID:   coinTypeID,
		CoinName:     coin.Name,
		DisplayNames: coin.DisplayNames,
		CoinLogo:     coin.Logo,
		CoinUnit:     coin.Unit,
		GoodMainCoin: goodMainCoin,
		Units:        units.String(),
		Incoming:     amount.String(),
	}
	if appPowerRental, ok := h.appPowerRentals[appGoodID]; ok {
		info.GoodQuantityUnit = appPowerRental.QuantityUnit
	}
	h.infos = append(h.infos, info)
}

func (h *goodProfitHandler) formalize() {
	profits := map[string]map[string]decimal.Decimal{}
	for _, order := range h.orders {
		good, ok := h.appGoods[order.AppGoodID]
		if !ok {
			continue
		}
		goodProfit, ok := profits[good.EntID]
		if !ok {
			goodProfit = map[string]decimal.Decimal{}
		}
		for _, statement := range h.statements[order.EntID] {
			coinProfit := goodProfit[statement.CoinTypeID]
			coinProfit = coinProfit.Add(decimal.RequireFromString(statement.Amount))
			goodProfit[statement.CoinTypeID] = coinProfit
		}
		profits[good.EntID] = goodProfit
	}

	for _, good := range h.appGoods {
		goodCoins, ok := h.goodCoins[good.GoodID]
		if !ok {
			continue
		}
		goodProfit, ok := profits[good.EntID]
		if !ok {
			for _, goodCoin := range goodCoins {
				h.formalizeProfit(good.EntID, goodCoin.CoinTypeID, goodCoin.Main, decimal.NewFromInt(0))
			}
			continue
		}
		for _, goodCoin := range goodCoins {
			coinProfit, ok := goodProfit[goodCoin.CoinTypeID]
			if !ok {
				h.formalizeProfit(good.EntID, goodCoin.CoinTypeID, goodCoin.Main, decimal.NewFromInt(0))
				continue
			}
			h.formalizeProfit(good.EntID, goodCoin.CoinTypeID, goodCoin.Main, coinProfit)
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
			h.orders[order.EntID] = order
		}
		offset += limit
	}
	return nil
}

func (h *goodProfitHandler) getPowerRentalOrders(ctx context.Context) error {
	orderIDs := func() (uids []string) {
		for orderID, order := range h.orders {
			switch order.GoodType {
			case goodtypes.GoodType_PowerRental:
			case goodtypes.GoodType_LegacyPowerRental:
			default:
				continue
			}
			uids = append(uids, orderID)
		}
		return
	}()
	powerRentalOrders, _, err := powerrentalordermwcli.GetPowerRentalOrders(ctx, &powerrentalordermwpb.Conds{
		AppID:    &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		OrderIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: orderIDs},
	}, 0, int32(len(orderIDs)))
	if err != nil {
		return err
	}
	h.powerRentalOrders = powerRentalOrders
	return nil
}

func (h *goodProfitHandler) getAppGoods(ctx context.Context) error {
	goods, total, err := appgoodmwcli.GetGoods(ctx, &appgoodmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}, h.Offset, h.Limit)
	if err != nil {
		return err
	}
	if len(goods) == 0 {
		return nil
	}
	h.total = total
	for _, good := range goods {
		h.appGoods[good.EntID] = good
	}
	return nil
}

func (h *goodProfitHandler) getAppPowerRentals(ctx context.Context) error {
	appGoodIDs := func() (_appGoodIDs []string) {
		for _, appGood := range h.appGoods {
			switch appGood.GoodType {
			case goodtypes.GoodType_PowerRental:
			case goodtypes.GoodType_LegacyPowerRental:
			default:
				continue
			}
			_appGoodIDs = append(_appGoodIDs, appGood.EntID)
		}
		return
	}()
	appPowerRentals, _, err := apppowerrentalmwcli.GetPowerRentals(ctx, &apppowerrentalmwpb.Conds{
		AppGoodIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: appGoodIDs},
	}, 0, int32(len(appGoodIDs)))
	if err != nil {
		return err
	}
	for _, appPowerRental := range appPowerRentals {
		h.appPowerRentals[appPowerRental.AppGoodID] = appPowerRental
	}
	return nil
}

func (h *goodProfitHandler) getAppCoins(ctx context.Context) error {
	for _, goodCoins := range h.goodCoins {
		for _, goodCoin := range goodCoins {
			if _, err := uuid.Parse(goodCoin.CoinTypeID); err != nil {
				continue
			}
			h.coinTypeIDs = append(h.coinTypeIDs, goodCoin.CoinTypeID)
		}
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

func (h *goodProfitHandler) getGoodCoins(ctx context.Context) error {
	goodIDs := func() (_goodIDs []string) {
		for _, appGood := range h.appGoods {
			_goodIDs = append(_goodIDs, appGood.GoodID)
		}
		return
	}()
	goodCoins, _, err := goodcoinmwcli.GetGoodCoins(ctx, &goodcoinmwpb.Conds{
		GoodIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: goodIDs},
	}, 0, int32(len(goodIDs)))
	if err != nil {
		return err
	}
	for _, goodCoin := range goodCoins {
		h.goodCoins[goodCoin.GoodID] = append(h.goodCoins[goodCoin.GoodID], goodCoin)
	}
	return nil
}

//nolint:dupl
func (h *goodProfitHandler) getStatements(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit
	conds := &statementmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		IOType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ledgertypes.IOType_Incoming)},
		IOSubTypes: &basetypes.Uint32SliceVal{Op: cruder.IN, Value: []uint32{
			uint32(ledgertypes.IOSubType_MiningBenefit),
			uint32(ledgertypes.IOSubType_SimulateMiningBenefit),
		}},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.coinTypeIDs},
	}
	if h.StartAt != nil {
		conds.StartAt = &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.StartAt}
	}
	if h.EndAt != nil {
		conds.EndAt = &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.EndAt}
	}

	for {
		statements, _, err := statementcli.GetStatements(ctx, conds, offset, limit)
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
				continue
			}
			order, ok := h.orders[e.OrderID]
			if !ok {
				continue
			}
			if order.AppGoodID != e.AppGoodID {
				continue
			}
			orderStatements, ok := h.statements[order.EntID]
			if !ok {
				orderStatements = []*statementmwpb.Statement{}
			}
			orderStatements = append(orderStatements, statement)
			h.statements[order.EntID] = orderStatements
		}
		offset += limit
	}
	return nil
}

func (h *Handler) GetGoodProfits(ctx context.Context) ([]*npool.GoodProfit, uint32, error) {
	handler := &goodProfitHandler{
		Handler:         h,
		appCoins:        map[string]*appcoinmwpb.Coin{},
		orders:          map[string]*ordermwpb.Order{},
		statements:      map[string][]*statementmwpb.Statement{},
		appGoods:        map[string]*appgoodmwpb.Good{},
		appPowerRentals: map[string]*apppowerrentalmwpb.PowerRental{},
		goodCoins:       map[string][]*goodcoinmwpb.GoodCoin{},
	}
	if err := h.CheckStartEndAt(); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppGoods(ctx); err != nil {
		return nil, 0, err
	}
	if len(handler.appGoods) == 0 {
		return nil, handler.total, nil
	}
	if err := handler.getAppPowerRentals(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getGoodCoins(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getPowerRentalOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getStatements(ctx); err != nil {
		return nil, 0, err
	}
	handler.formalize()
	return handler.infos, handler.total, nil
}
