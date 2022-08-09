package ledger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermgrprofitcli "github.com/NpoolPlatform/ledger-manager/pkg/client/profit"
	ledgermgrdetailpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/detail"
	ledgermgrprofitpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/profit"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	coininfopb "github.com/NpoolPlatform/message/npool/coininfo"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	goodscli "github.com/NpoolPlatform/cloud-hashing-goods/pkg/client"
	goodspb "github.com/NpoolPlatform/message/npool/cloud-hashing-goods"

	ordercli "github.com/NpoolPlatform/cloud-hashing-order/pkg/client"
	orderpb "github.com/NpoolPlatform/message/npool/cloud-hashing-order"

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

	coins, err := coininfocli.GetCoinInfos(ctx, cruder.NewFilterConds())
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*coininfopb.CoinInfo{}
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

	coins, err := coininfocli.GetCoinInfos(ctx, cruder.NewFilterConds())
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*coininfopb.CoinInfo{}
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

	coins, err := coininfocli.GetCoinInfos(ctx, cruder.NewFilterConds())
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*coininfopb.CoinInfo{}
	for _, coin := range coins {
		coinMap[coin.ID] = coin
	}

	goods, err := goodscli.GetGoods(ctx)
	if err != nil {
		return nil, 0, err
	}

	goodMap := map[string]*goodspb.GoodInfo{}
	for _, good := range goods {
		goodMap[good.ID] = good
	}

	// TODO: offset / limit is actually not for orders here, and not used
	orders, err := ordercli.GetUserOrders(ctx, appID, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	orderMap := map[string]*orderpb.Order{}
	for _, order := range orders {
		orderMap[order.ID] = order
	}

	type extra struct {
		BenefitDate string
		OrderID     string
	}

	infos := map[string]*npool.GoodProfit{}
	total := uint32(0)

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

		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid coin")
		}

		e := extra{}
		err := json.Unmarshal([]byte(info.IOExtra), &e)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid io extra")
		}

		order, ok := orderMap[e.OrderID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid order")
		}

		good, ok := goodMap[order.GoodID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid good")
		}

		gp, ok := infos[order.GoodID]
		if !ok {
			gp = &npool.GoodProfit{
				CoinTypeID: info.CoinTypeID,
				CoinName:   coin.Name,
				CoinLogo:   coin.Logo,
				CoinUnit:   coin.Unit,
				GoodID:     order.GoodID,
				GoodName:   good.Title,
				GoodUnit:   good.Unit,
				Units:      0,
				Incoming:   decimal.NewFromInt(0).String(),
			}
			total += 1
		}

		if info.IOSubType == ledgermgrdetailpb.IOSubType_MiningBenefit {
			gp.Incoming = decimal.RequireFromString(gp.Incoming).
				Add(decimal.RequireFromString(info.Amount)).
				String()
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
