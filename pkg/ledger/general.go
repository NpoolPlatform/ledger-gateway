package ledger

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermgrgeneralcli "github.com/NpoolPlatform/ledger-manager/pkg/client/general"
	ledgermgrgeneralpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/general"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	coininfopb "github.com/NpoolPlatform/message/npool/coininfo"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"
)

func GetGenerals(ctx context.Context, appID, userID string, offset, limit int32) ([]*npool.General, uint32, error) {
	coins, total, err := coininfocli.GetCoinInfosV2(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(coins) == 0 {
		return nil, 0, nil
	}

	coinTypeIDs := []string{}
	for _, coin := range coins {
		coinTypeIDs = append(coinTypeIDs, coin.ID)
	}

	conds := &ledgermgrgeneralpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: userID,
		},
		CoinTypeIDs: &commonpb.StringSliceVal{
			Op:    cruder.IN,
			Value: coinTypeIDs,
		},
	}

	infos, _, err := ledgermgrgeneralcli.GetGenerals(ctx, conds, 0, limit)
	if err != nil {
		return nil, 0, err
	}

	generalMap := map[string]*ledgermgrgeneralpb.General{}
	for _, general := range infos {
		generalMap[general.CoinTypeID] = general
	}

	generals := []*npool.General{}
	for _, coin := range coins {
		general, ok := generalMap[coin.ID]
		if ok {
			generals = append(generals, &npool.General{
				CoinTypeID: coin.ID,
				CoinName:   coin.Name,
				CoinLogo:   coin.Logo,
				CoinUnit:   coin.Unit,
				Incoming:   general.Incoming,
				Locked:     general.Locked,
				Outcoming:  general.Outcoming,
				Spendable:  general.Spendable,
			})
		} else {
			generals = append(generals, &npool.General{
				CoinTypeID: coin.ID,
				CoinName:   coin.Name,
				CoinLogo:   coin.Logo,
				CoinUnit:   coin.Unit,
				Incoming:   decimal.NewFromInt(0).String(),
				Locked:     decimal.NewFromInt(0).String(),
				Outcoming:  decimal.NewFromInt(0).String(),
				Spendable:  decimal.NewFromInt(0).String(),
			})
		}
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
