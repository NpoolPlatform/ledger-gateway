package ledger

import (
	"context"
	"fmt"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermgrgeneralcli "github.com/NpoolPlatform/ledger-manager/pkg/client/general"
	ledgermgrprofitcli "github.com/NpoolPlatform/ledger-manager/pkg/client/profit"
	ledgermgrgeneralpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/general"
	ledgermgrprofitpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/profit"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	coininfopb "github.com/NpoolPlatform/message/npool/coininfo"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"
)

func GetGenerals(ctx context.Context, appID, userID string, offset, limit int32) ([]*npool.General, uint32, error) {
	conds := &ledgermgrgeneralpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: userID,
		},
	}

	infos, total, err := ledgermgrgeneralcli.GetGenerals(ctx, conds, offset, limit)
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

	generals := []*npool.General{}
	for _, info := range infos {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid coin")
		}

		generals = append(generals, &npool.General{
			CoinTypeID: info.CoinTypeID,
			CoinName:   coin.Name,
			CoinLogo:   coin.Logo,
			CoinUnit:   coin.Unit,
			Incoming:   info.Incoming,
			Locked:     info.Locked,
			Outcoming:  info.Outcoming,
			Spendable:  info.Spendable,
		})
	}

	return generals, total, nil
}

func GetIntervalGenerals(
	ctx context.Context, appID, userID string, start, end uint32, offset, limit int32,
) (
	[]*npool.General, uint32, error,
) {
	generals, total, err := ledgermwcli.GetIntervalGenerals(ctx, appID, userID, start, end, offset, limit)
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

	infos := []*npool.General{}
	for _, info := range generals {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid coin")
		}

		infos = append(infos, &npool.General{
			CoinTypeID: info.CoinTypeID,
			CoinName:   coin.Name,
			CoinLogo:   coin.Logo,
			CoinUnit:   coin.Unit,
			Incoming:   info.Incoming,
			Locked:     info.Locked,
			Outcoming:  info.Outcoming,
			Spendable:  info.Spendable,
		})
	}

	return infos, total, nil
}

func GetDetails(ctx context.Context, appID, userID string, start, end uint32, offset, limit int32) ([]*npool.Detail, uint32, error) {
	details, total, err := ledgermwcli.GetIntervalDetails(ctx, appID, userID, start, end, offset, limit)
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

	infos := []*npool.Detail{}
	for _, info := range details {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid coin")
		}

		infos = append(infos, &npool.Detail{
			CoinTypeID: info.CoinTypeID,
			CoinName:   coin.Name,
			CoinLogo:   coin.Logo,
			CoinUnit:   coin.Unit,
			IOType:     info.IOType.String(),
			IOSubType:  info.IOSubType.String(),
			Amount:     info.Amount,
			IOExtra:    info.IOExtra,
		})
	}

	return infos, total, nil
}

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
	details, total, err := ledgermwcli.GetIntervalDetails(ctx, appID, userID, start, end, offset, limit)
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

	infos := []*npool.Profit{}
	for _, info := range details {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid coin")
		}

		infos = append(infos, &npool.Profit{
			CoinTypeID: info.CoinTypeID,
			CoinName:   coin.Name,
			CoinLogo:   coin.Logo,
			CoinUnit:   coin.Unit,
			Incoming:   info.Amount,
		})
	}

	return infos, total, nil
}
