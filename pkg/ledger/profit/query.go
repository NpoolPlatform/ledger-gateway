package profit

import (
	"context"
	"encoding/json"
	"fmt"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	goodscli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	statementhandler "github.com/NpoolPlatform/ledger-gateway/pkg/ledger/statement"
	profitmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/profit"
	statementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"
	ledgerpb "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	goodsmgrpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/good"
	goodspb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	profitpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/profit"
	"github.com/NpoolPlatform/message/npool/ledger/mw/v2/statement"
	orderpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order"
	orderstatemgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func (h *Handler) setConds() *profitpb.Conds {
	conds := &profitpb.Conds{}
	if h.AppID != nil {
		conds.AppID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID}
	}
	if h.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}
	return conds
}

// Export In Frontend
func (h *Handler) GetMiningRewards(ctx context.Context) ([]*npool.MiningReward, uint32, error) {
	statementHandler := &statementhandler.Handler{
		Handler: h.Handler,
	}
	statements, total, err := statementHandler.GetStatements(ctx)
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

// Mining Summary
func (h *Handler) GetProfits(ctx context.Context) ([]*npool.Profit, uint32, error) {
	profits, total, err := profitmwcli.GetProfits(ctx, h.setConds(), h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}
	if len(profits) == 0 {
		return nil, total, nil
	}

	coinTypeIDs := []string{}
	for _, profit := range profits {
		if _, err := uuid.Parse(profit.CoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, profit.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
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

	coinMap := map[string]*appcoinmwpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.CoinTypeID] = coin
	}

	infos := []*npool.Profit{}
	for _, info := range profits {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			logger.Sugar().Warn("app coin not exist continue, cointypeid(%v)", info.CoinTypeID)
			continue
		}
		infos = append(infos, &npool.Profit{
			CoinTypeID:   info.CoinTypeID,
			CoinName:     coin.Name,
			DisplayNames: coin.DisplayNames,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			Incoming:     info.Incoming,
		})
	}

	return infos, total, nil
}

// Mining Interval Profit
func (h *Handler) GetIntervalProfits(ctx context.Context) ([]*npool.Profit, uint32, error) {
	ioType := ledgerpb.IOType_Incoming
	ioSubType := ledgerpb.IOSubType_MiningBenefit

	statements := []*statement.Statement{}
	var total uint32
	ofs := int32(0)
	lim := h.Limit
	for {
		st, _total, err := statementcli.GetStatements(ctx, &statement.Conds{
			AppID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID:    &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			IOType:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ioType)},
			IOSubType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ioSubType)},
			StartAt:   &basetypes.Uint32Val{Op: cruder.GT, Value: h.StartAt},
			EndAt:     &basetypes.Uint32Val{Op: cruder.LT, Value: h.EndAt},
		}, h.Offset, h.Limit)
		if err != nil {
			return nil, 0, err
		}
		total = _total
		if len(st) == 0 {
			break
		}
		statements = append(statements, st...)
		ofs += lim
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

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
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

	coinMap := map[string]*appcoinmwpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.CoinTypeID] = coin
	}

	infos := map[string]*npool.Profit{}
	for _, info := range statements {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			logger.Sugar().Errorf("invalid coin, %v", info.CoinTypeID)
			continue
		}

		p, ok := infos[info.CoinTypeID]
		if !ok {
			p = &npool.Profit{
				CoinTypeID:   info.CoinTypeID,
				CoinName:     coin.Name,
				DisplayNames: coin.DisplayNames,
				CoinLogo:     coin.Logo,
				CoinUnit:     coin.Unit,
				Incoming:     decimal.NewFromInt(0).String(),
			}
		}

		p.Incoming = decimal.RequireFromString(p.Incoming).
			Add(decimal.RequireFromString(info.Amount)).
			String()

		infos[info.CoinTypeID] = p
	}

	profits := []*npool.Profit{}
	for _, info := range infos {
		profits = append(profits, info)
	}

	return profits, total, nil
}

// Good Card
// nolint
func (h *Handler) GetGoodProfits(ctx context.Context) ([]*npool.GoodProfit, uint32, error) {
	statements := []*statement.Statement{}
	ofs := int32(0)
	lim := h.Limit
	for {
		st, _, err := statementcli.GetStatements(ctx, &statement.Conds{
			AppID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			StartAt: &basetypes.Uint32Val{Op: cruder.GT, Value: h.StartAt},
			EndAt:   &basetypes.Uint32Val{Op: cruder.LT, Value: h.EndAt},
		}, h.Offset, h.Limit)
		if err != nil {
			return nil, 0, err
		}
		if len(st) == 0 {
			break
		}
		statements = append(statements, st...)
		ofs += lim
	}
	if len(statements) == 0 {
		return nil, 0, nil
	}

	orders := []*ordermwpb.Order{}
	ofs = 0
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

	type extra struct {
		BenefitDate string
		OrderID     string
	}

	infos := map[string]*npool.GoodProfit{}
	total := uint32(0)

	profitOrderMap := map[string]struct{}{}

	goodIDs := []string{}
	for _, val := range orders {
		goodIDs = append(goodIDs, val.GetGoodID())
	}

	goods, _, err := goodscli.GetGoods(ctx, &goodsmgrpb.Conds{
		IDs: &commonpb.StringSliceVal{
			Op:    cruder.IN,
			Value: goodIDs,
		},
	}, 0, int32(len(goodIDs)))
	if err != nil {
		return nil, 0, err
	}

	goodMap := map[string]*goodspb.Good{}
	for _, good := range goods {
		goodMap[good.ID] = good
	}

	coinTypeIDs := []string{}
	for _, val := range statements {
		if _, err := uuid.Parse(val.CoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}

	for _, val := range goods {
		if _, err := uuid.Parse(val.CoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
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

	coinMap := map[string]*appcoinmwpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.CoinTypeID] = coin
	}

	for _, info := range statements {
		if info.IOType != ledgerpb.IOType_Incoming {
			continue
		}
		switch info.IOSubType {
		case ledgerpb.IOSubType_MiningBenefit:
		case ledgerpb.IOSubType_Payment:
		default:
			continue
		}

		e := extra{}
		err := json.Unmarshal([]byte(info.IOExtra), &e)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid io extra")
		}

		order, ok := orderMap[e.OrderID]
		if !ok {
			logger.Sugar().Warnw("GetGoodProfits", "ID", info.ID, "OrderID", e.OrderID)
			continue
		}

		switch order.OrderState {
		case orderstatemgrpb.OrderState_Paid:
		case orderstatemgrpb.OrderState_InService:
		case orderstatemgrpb.OrderState_Expired:
		default:
			continue
		}

		good, ok := goodMap[order.GoodID]
		if !ok {
			logger.Sugar().Warnw("GetGoodProfits", "ID", info.ID, "GoodID", order.GoodID)
			continue
		}

		coin, ok := coinMap[good.CoinTypeID]
		if !ok {
			logger.Sugar().Warnw("GetGoodProfits", "ID", info.ID, "CoinTypeID", good.CoinTypeID, "GoodID", order.GoodID)
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
			total += 1
		}

		if info.IOSubType == ledgerpb.IOSubType_MiningBenefit {
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

	for _, order := range orders {
		if _, ok := profitOrderMap[order.ID]; ok {
			continue
		}

		switch order.OrderState {
		case orderstatemgrpb.OrderState_Paid:
		case orderstatemgrpb.OrderState_InService:
		case orderstatemgrpb.OrderState_Expired:
		default:
			continue
		}

		good, ok := goodMap[order.GoodID]
		if !ok {
			logger.Sugar().Warn("good %v not exist", order.GoodID)
			continue
		}

		coin, ok := coinMap[good.CoinTypeID]
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
			total += 1
		}

		gp.Units = decimal.
			RequireFromString(gp.Units).
			Add(decimal.RequireFromString(order.Units)).
			String()
		infos[order.GoodID] = gp
	}

	profits := []*npool.GoodProfit{}
	for _, info := range infos {
		profits = append(profits, info)
	}

	return profits, total, nil
}
