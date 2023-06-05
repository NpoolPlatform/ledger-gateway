package ledger

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermgrgeneralcli "github.com/NpoolPlatform/ledger-manager/pkg/client/general"
	ledgermgrgeneralpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/general"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	commonpb "github.com/NpoolPlatform/message/npool"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"

	"github.com/google/uuid"
)

func GetGenerals(ctx context.Context, appID, userID string, offset, limit int32) ([]*npool.General, uint32, error) {
	coins, total, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
	}, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(coins) == 0 {
		return nil, 0, nil
	}

	coinTypeIDs := []string{}
	for _, coin := range coins {
		coinTypeIDs = append(coinTypeIDs, coin.CoinTypeID)
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
		general, ok := generalMap[coin.CoinTypeID]
		if ok {
			generals = append(generals, &npool.General{
				CoinTypeID:   coin.CoinTypeID,
				CoinName:     coin.Name,
				DisplayNames: coin.DisplayNames,
				CoinLogo:     coin.Logo,
				CoinUnit:     coin.Unit,
				CoinDisabled: coin.Disabled,
				CoinDisplay:  coin.Display,
				Incoming:     general.Incoming,
				Locked:       general.Locked,
				Outcoming:    general.Outcoming,
				Spendable:    general.Spendable,
			})
		} else {
			generals = append(generals, &npool.General{
				CoinTypeID:   coin.CoinTypeID,
				CoinName:     coin.Name,
				DisplayNames: coin.DisplayNames,
				CoinLogo:     coin.Logo,
				CoinUnit:     coin.Unit,
				CoinDisabled: coin.Disabled,
				CoinDisplay:  coin.Display,
				Incoming:     decimal.NewFromInt(0).String(),
				Locked:       decimal.NewFromInt(0).String(),
				Outcoming:    decimal.NewFromInt(0).String(),
				Spendable:    decimal.NewFromInt(0).String(),
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

	ids := []string{}
	for _, g := range generals {
		if _, err := uuid.Parse(g.CoinTypeID); err != nil {
			continue
		}
		ids = append(ids, g.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		CoinTypeIDs: &basetypes.StringSliceVal{
			Op:    cruder.IN,
			Value: ids,
		},
	}, 0, int32(len(ids)))
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*appcoinmwpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.CoinTypeID] = coin
	}

	infos := []*npool.General{}
	for _, info := range generals {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			return nil, 0, fmt.Errorf("invalid coin")
		}

		infos = append(infos, &npool.General{
			CoinTypeID:   info.CoinTypeID,
			CoinName:     coin.Name,
			DisplayNames: coin.DisplayNames,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			Incoming:     info.Incoming,
			Locked:       info.Locked,
			Outcoming:    info.Outcoming,
			Spendable:    info.Spendable,
		})
	}

	return infos, total, nil
}

func GetAppGenerals(ctx context.Context, appID string, offset, limit int32) ([]*npool.General, uint32, error) {
	conds := &ledgermgrgeneralpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
	}

	infos, total, err := ledgermgrgeneralcli.GetGenerals(ctx, conds, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	if len(infos) == 0 {
		return nil, 0, nil
	}

	userIDs := []string{}
	for _, info := range infos {
		userIDs = append(userIDs, info.UserID)
	}

	users, _, err := usermwcli.GetUsers(ctx, &usermwpb.Conds{
		IDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: userIDs},
	}, 0, int32(len(userIDs)))
	if err != nil {
		return nil, 0, fmt.Errorf("fail get users: %v", err)
	}

	userMap := map[string]*usermwpb.User{}
	for _, user := range users {
		userMap[user.ID] = user
	}

	ids := []string{}
	for _, info := range infos {
		if _, err := uuid.Parse(info.CoinTypeID); err != nil {
			continue
		}
		ids = append(ids, info.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		IDs: &basetypes.StringSliceVal{
			Op:    cruder.IN,
			Value: ids,
		},
	}, 0, int32(len(ids)))
	if err != nil {
		return nil, 0, fmt.Errorf("fail get coins: %v", err)
	}

	coinMap := map[string]*appcoinmwpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.CoinTypeID] = coin
	}

	generals := []*npool.General{}
	for _, general := range infos {
		user, ok := userMap[general.UserID]
		if !ok {
			continue
		}
		coin, ok := coinMap[general.CoinTypeID]
		if !ok {
			continue
		}
		generals = append(generals, &npool.General{
			CoinTypeID:   coin.CoinTypeID,
			CoinName:     coin.Name,
			DisplayNames: coin.DisplayNames,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			Incoming:     general.Incoming,
			Locked:       general.Locked,
			Outcoming:    general.Outcoming,
			Spendable:    general.Spendable,
			UserID:       general.UserID,
			PhoneNO:      user.PhoneNO,
			EmailAddress: user.EmailAddress,
		})
	}
	return generals, total, nil
}
