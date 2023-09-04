package profit

import (
	"context"
	"encoding/json"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	statementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	types "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	orderpb "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	goodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	statementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	"github.com/shopspring/decimal"
)

type profitHandler struct {
	*Handler
	statements    []*statementmwpb.Statement
	orders        map[string]*ordermwpb.Order
	appcoins      map[string]*appcoinmwpb.Coin
	goods         map[string]*goodmwpb.Good
	miningRewards []*npool.MiningReward
	infos         []*npool.Profit
	goodProfits   []*npool.GoodProfit
}

//nolint
func (h *profitHandler) getAppCoins(ctx context.Context) error {
	coinTypeIDs := []string{}
	for _, val := range h.statements {
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return err
	}
	for _, coin := range coins {
		h.appcoins[coin.CoinTypeID] = coin
	}
	return nil
}

func (h *profitHandler) getOrders(ctx context.Context) error {
	ofs := int32(0)
	lim := int32(100) //nolint
	infos := []*ordermwpb.Order{}
	for {
		orders, _, err := ordermwcli.GetOrders(ctx, &ordermwpb.Conds{
			AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		}, ofs, lim)
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			break
		}

		infos = append(infos, orders...)
		ofs += lim
	}
	for _, order := range infos {
		h.orders[order.ID] = order
	}
	return nil
}

// Export In Frontend
func (h *profitHandler) miningRewardsFormalize() {
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
			continue
		}

		coin, ok := h.appcoins[val.CoinTypeID]
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

		rewardAmount, err := decimal.NewFromString(val.Amount)
		if err != nil {
			continue
		}
		units, err := decimal.NewFromString(order.Units)
		if err != nil {
			continue
		}

		h.miningRewards = append(h.miningRewards, &npool.MiningReward{
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
	ioSubType := types.IOSubType_MiningBenefit
	total := uint32(0)
	offset := int32(0)
	limit := int32(1000) //nolint
	statements := []*statementmwpb.Statement{}
	for {
		infos, _total, err := statementcli.GetStatements(ctx, &statementmwpb.Conds{
			AppID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID:    &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			IOSubType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ioSubType)},
			StartAt:   &basetypes.Uint32Val{Op: cruder.EQ, Value: h.StartAt},
			EndAt:     &basetypes.Uint32Val{Op: cruder.EQ, Value: h.EndAt},
		}, offset, limit)
		if err != nil {
			return nil, total, err
		}
		total = _total
		if len(infos) == 0 {
			break
		}
		statements = append(statements, infos...)
		offset += limit
	}
	if len(statements) == 0 {
		return nil, 0, nil
	}

	handler := &profitHandler{
		Handler:    h,
		statements: statements,
		appcoins:   map[string]*appcoinmwpb.Coin{},
		orders:     map[string]*ordermwpb.Order{},
	}

	if err := handler.getAppCoins(ctx); err != nil {
		return nil, total, err
	}
	if err := handler.getOrders(ctx); err != nil {
		return nil, total, err
	}

	handler.miningRewardsFormalize()
	return handler.miningRewards, total, nil
}

func (h *profitHandler) intervalProfitsFormalize() {
	infos := map[string]*npool.Profit{}
	for _, val := range h.statements {
		coin, ok := h.appcoins[val.CoinTypeID]
		if !ok {
			continue
		}

		p, ok := infos[val.CoinTypeID]
		if !ok {
			p = &npool.Profit{
				CoinTypeID:   val.CoinTypeID,
				CoinName:     coin.Name,
				DisplayNames: coin.DisplayNames,
				CoinLogo:     coin.Logo,
				CoinUnit:     coin.Unit,
				Incoming:     decimal.NewFromInt(0).String(),
			}
		}

		p.Incoming = decimal.RequireFromString(p.Incoming).
			Add(decimal.RequireFromString(val.Amount)).
			String()

		infos[val.CoinTypeID] = p
	}

	for _, info := range infos {
		h.infos = append(h.infos, info)
	}
}

// Mining Interval Profit
func (h *Handler) GetIntervalProfits(ctx context.Context) ([]*npool.Profit, uint32, error) {
	ioType := types.IOType_Incoming
	ioSubType := types.IOSubType_MiningBenefit

	statements := []*statementmwpb.Statement{}
	var total uint32
	offset := h.Offset
	limit := h.Limit
	for {
		sts, _total, err := statementcli.GetStatements(ctx, &statementmwpb.Conds{
			AppID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID:    &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			IOType:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ioType)},
			IOSubType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ioSubType)},
			StartAt:   &basetypes.Uint32Val{Op: cruder.EQ, Value: h.StartAt},
			EndAt:     &basetypes.Uint32Val{Op: cruder.EQ, Value: h.EndAt},
		}, offset, limit)
		if err != nil {
			return nil, 0, err
		}
		total = _total

		if len(sts) == 0 {
			break
		}
		statements = append(statements, sts...)
		offset += limit
	}
	if len(statements) == 0 {
		return nil, 0, nil
	}

	handler := &profitHandler{
		Handler:    h,
		statements: statements,
		appcoins:   map[string]*appcoinmwpb.Coin{},
	}

	if err := handler.getAppCoins(ctx); err != nil {
		return nil, total, err
	}

	handler.intervalProfitsFormalize()
	return handler.infos, total, nil
}

func (h *profitHandler) getGoods(ctx context.Context) error {
	ids := []string{}
	for _, order := range h.orders {
		ids = append(ids, order.GoodID)
	}

	goods, _, err := goodmwcli.GetGoods(ctx, &goodmwpb.Conds{
		IDs: &basetypes.StringSliceVal{
			Op:    cruder.IN,
			Value: ids,
		},
	}, 0, int32(len(ids)))
	if err != nil {
		return err
	}

	for _, good := range goods {
		h.goods[good.ID] = good
	}
	return nil
}

func (h *profitHandler) goodProfitsFormalize() { //nolint
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
			continue
		}

		coin, ok := h.appcoins[good.CoinTypeID]
		if !ok {
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

		coin, ok := h.appcoins[good.CoinTypeID]
		if !ok {
			logger.Sugar().Warn("coin not exist")
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
		h.goodProfits = append(h.goodProfits, info)
	}
}

func (h *Handler) GetGoodProfits(ctx context.Context) ([]*npool.GoodProfit, uint32, error) {
	total := uint32(0)
	statements := []*statementmwpb.Statement{}
	offset := int32(0)
	limit := h.Limit
	ioType := types.IOType_Incoming
	for {
		sts, _total, err := statementcli.GetStatements(ctx, &statementmwpb.Conds{
			AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			IOType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ioType)},
			IOSubTypes: &basetypes.Uint32SliceVal{Op: cruder.IN, Value: []uint32{
				uint32(types.IOSubType_MiningBenefit), uint32(types.IOSubType_Payment),
			}},
			StartAt: &basetypes.Uint32Val{Op: cruder.EQ, Value: h.StartAt},
			EndAt:   &basetypes.Uint32Val{Op: cruder.EQ, Value: h.EndAt},
		}, offset, limit)
		if err != nil {
			return nil, 0, err
		}
		total = _total
		if len(sts) == 0 {
			break
		}
		statements = append(statements, sts...)
		offset += limit
	}
	if len(statements) == 0 {
		return nil, 0, nil
	}

	handler := &profitHandler{
		Handler:    h,
		statements: statements,
		appcoins:   map[string]*appcoinmwpb.Coin{},
		orders:     map[string]*ordermwpb.Order{},
	}

	if err := handler.getOrders(ctx); err != nil {
		return nil, total, err
	}
	if err := handler.getGoods(ctx); err != nil {
		return nil, total, err
	}

	coinTypeIDs := []string{}
	for _, val := range statements {
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}
	for _, val := range handler.goods {
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return nil, 0, err
	}

	for _, coin := range coins {
		handler.appcoins[coin.CoinTypeID] = coin
	}
	handler.goodProfitsFormalize()
	return handler.goodProfits, total, nil
}
