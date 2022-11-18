package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	appcoinpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/appcoin"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	"github.com/shopspring/decimal"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermgrprofitcli "github.com/NpoolPlatform/ledger-manager/pkg/client/profit"
	ledgermgrdetailpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/detail"
	ledgermgrprofitpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/profit"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	orderstatemgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order"

	coininfocli "github.com/NpoolPlatform/chain-middleware/pkg/client/appcoin"

	goodscli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	goodspb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"

	goodsmgrpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/good"

	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"
)

func GetProfits(ctx context.Context, appID, userID string, offset, limit int32) ([]*npool.Profit, uint32, error) {
	conds := &ledgermgrprofitpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: userID,
		},
	}

	infos, total, err := ledgermgrprofitcli.GetProfits(ctx, conds, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(infos) == 0 {
		return nil, total, nil
	}

	coinTypeIDs := []string{}
	for _, val := range infos {
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}

	coins, _, err := coininfocli.GetCoins(ctx, &appcoinpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		CoinTypeIDs: &commonpb.StringSliceVal{
			Op:    cruder.EQ,
			Value: coinTypeIDs,
		},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*appcoinpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.ID] = coin
	}

	profits := []*npool.Profit{}
	for _, info := range infos {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid coin")
		}

		profits = append(profits, &npool.Profit{
			CoinTypeID: info.CoinTypeID,
			CoinName:   coin.Name,
			CoinLogo:   coin.Logo,
			CoinUnit:   coin.Unit,
			Incoming:   info.Incoming,
		})
	}

	return profits, total, nil
}

func GetIntervalProfits(
	ctx context.Context, appID, userID string, start, end uint32, offset, limit int32,
) (
	[]*npool.Profit, uint32, error,
) {
	// TODO: move to middleware with aggregate
	details := []*ledgermgrdetailpb.Detail{}
	ofs := int32(0)
	lim := limit

	if lim == 0 {
		lim = 1000
	}

	for {
		ds, _, err := ledgermwcli.GetIntervalDetails(
			ctx, appID, userID, start, end, ofs, lim,
		)
		if err != nil {
			return nil, 0, err
		}
		if len(ds) == 0 {
			break
		}

		details = append(details, ds...)
		ofs += lim
	}

	coinTypeIDs := []string{}
	for _, val := range details {
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}

	coins, _, err := coininfocli.GetCoins(ctx, &appcoinpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		CoinTypeIDs: &commonpb.StringSliceVal{
			Op:    cruder.EQ,
			Value: coinTypeIDs,
		},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*appcoinpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.ID] = coin
	}

	infos := map[string]*npool.Profit{}
	total := uint32(0)

	for _, info := range details {
		if info.IOType != ledgermgrdetailpb.IOType_Incoming {
			continue
		}
		if info.IOSubType != ledgermgrdetailpb.IOSubType_MiningBenefit {
			continue
		}

		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid coin")
		}

		p, ok := infos[info.CoinTypeID]
		if !ok {
			p = &npool.Profit{
				CoinTypeID: info.CoinTypeID,
				CoinName:   coin.Name,
				CoinLogo:   coin.Logo,
				CoinUnit:   coin.Unit,
				Incoming:   decimal.NewFromInt(0).String(),
			}
			total += 1
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

// nolint
func GetGoodProfits(
	ctx context.Context, appID, userID string, start, end uint32, offset, limit int32,
) (
	[]*npool.GoodProfit, uint32, error,
) {
	// TODO: move to middleware with aggregate
	details := []*ledgermgrdetailpb.Detail{}
	ofs := int32(0)
	lim := limit

	for {
		ds, _, err := ledgermwcli.GetIntervalDetails(
			ctx, appID, userID, start, end, ofs, lim,
		)
		if err != nil {
			return nil, 0, err
		}
		if len(ds) == 0 {
			break
		}

		details = append(details, ds...)
		ofs += lim
	}

	coinTypeIDs := []string{}
	for _, val := range details {
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}

	coins, _, err := coininfocli.GetCoins(ctx, &appcoinpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		CoinTypeIDs: &commonpb.StringSliceVal{
			Op:    cruder.EQ,
			Value: coinTypeIDs,
		},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*appcoinpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.ID] = coin
	}

	orders := []*ordermwpb.Order{}
	ofs = 0

	for {
		ords, _, err := ordermwcli.GetOrders(ctx, &ordermwpb.Conds{
			AppID: &commonpb.StringVal{
				Op:    cruder.EQ,
				Value: appID,
			},
			UserID: &commonpb.StringVal{
				Op:    cruder.EQ,
				Value: userID,
			},
		}, ofs, limit)
		if err != nil {
			return nil, 0, err
		}
		if len(ords) == 0 {
			break
		}

		orders = append(orders, ords...)

		ofs += limit
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

	for _, info := range details {
		if info.IOType != ledgermgrdetailpb.IOType_Incoming {
			continue
		}
		switch info.IOSubType {
		case ledgermgrdetailpb.IOSubType_MiningBenefit:
		case ledgermgrdetailpb.IOSubType_Payment:
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
			logger.Sugar().Warn("order not exist continue")
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
			logger.Sugar().Warn("good not exist continue")
			continue
		}

		coin, ok := coinMap[good.CoinTypeID]
		if !ok {
			logger.Sugar().Warn("coin not exist continue")
			continue
		}

		gp, ok := infos[order.GoodID]
		if !ok {
			gp = &npool.GoodProfit{
				CoinTypeID:            good.CoinTypeID,
				CoinName:              coin.Name,
				CoinLogo:              coin.Logo,
				CoinUnit:              coin.Unit,
				GoodID:                order.GoodID,
				GoodName:              good.Title,
				GoodUnit:              good.Unit,
				GoodServicePeriodDays: uint32(good.DurationDays),
				Units:                 0,
				Incoming:              decimal.NewFromInt(0).String(),
			}
			total += 1
		}

		if info.IOSubType == ledgermgrdetailpb.IOSubType_MiningBenefit {
			gp.Incoming = decimal.RequireFromString(gp.Incoming).
				Add(decimal.RequireFromString(info.Amount)).
				String()
		}

		gp.Units += order.Units

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
			logger.Sugar().Warn("good not exist continue")
			continue
		}

		coin, ok := coinMap[good.CoinTypeID]
		if !ok {
			logger.Sugar().Warn("good not exist continue")
			continue
		}

		gp, ok := infos[order.GoodID]
		if !ok {
			gp = &npool.GoodProfit{
				CoinTypeID:            good.CoinTypeID,
				CoinName:              coin.Name,
				CoinLogo:              coin.Logo,
				CoinUnit:              coin.Unit,
				GoodID:                order.GoodID,
				GoodName:              good.Title,
				GoodUnit:              good.Unit,
				GoodServicePeriodDays: uint32(good.DurationDays),
				Units:                 0,
				Incoming:              decimal.NewFromInt(0).String(),
			}
			total += 1
		}

		gp.Units += order.Units
		infos[order.GoodID] = gp
	}

	profits := []*npool.GoodProfit{}
	for _, info := range infos {
		profits = append(profits, info)
	}

	return profits, total, nil
}
