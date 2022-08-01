package ledger

import (
	"context"
	"fmt"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermgrcli "github.com/NpoolPlatform/ledger-manager/pkg/client/general"
	ledgermgrpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/general"

	coininfopb "github.com/NpoolPlatform/message/npool/coininfo"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"
)

func GetGenerals(ctx context.Context, appID, userID string, offset, limit int32) ([]*npool.General, uint32, error) {
	conds := &ledgermgrpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: userID,
		},
	}

	infos, total, err := ledgermgrcli.GetGenerals(ctx, conds, offset, limit)
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
	return nil, 0, fmt.Errorf("NOT IMPLEMENTED")
}

func GetDetails(ctx context.Context, appID, userID string, start, end uint32, offset, limit int32) ([]*npool.Detail, uint32, error) {
	return nil, 0, fmt.Errorf("NOT IMPLEMENTED")
}

func GetProfits(ctx context.Context, appID, userID string, offset, limit int32) ([]*npool.General, uint32, error) {
	return nil, 0, fmt.Errorf("NOT IMPLEMENTED")
}

func GetIntervalProfits(ctx context.Context, appID, userID string, offset, limit int32) ([]*npool.General, uint32, error) {
	return nil, 0, fmt.Errorf("NOT IMPLEMENTED")
}
