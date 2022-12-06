package ledger

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	commonpb "github.com/NpoolPlatform/message/npool"
	appusermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	appcoinpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/appcoin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	ledgermgrdetailpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/detail"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	coininfocli "github.com/NpoolPlatform/chain-middleware/pkg/client/appcoin"
	ledgermgrdetailcli "github.com/NpoolPlatform/ledger-manager/pkg/client/detail"
	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
)

func GetDetails(ctx context.Context, appID, userID string, start, end uint32, offset, limit int32) ([]*npool.Detail, uint32, error) {
	details, total, err := ledgermwcli.GetIntervalDetails(ctx, appID, userID, start, end, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(details) == 0 {
		return nil, total, nil
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
			Op:    cruder.IN,
			Value: coinTypeIDs,
		},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*appcoinpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.CoinTypeID] = coin
	}

	infos := []*npool.Detail{}
	for _, info := range details {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			logger.Sugar().Warnw("GetDetails", "app coin not exist", "appID", appID, "user_id", userID)
			continue
		}

		infos = append(infos, &npool.Detail{
			CoinTypeID: info.CoinTypeID,
			CoinName:   coin.Name,
			CoinLogo:   coin.Logo,
			CoinUnit:   coin.Unit,
			IOType:     info.IOType,
			IOSubType:  info.IOSubType,
			Amount:     info.Amount,
			IOExtra:    info.IOExtra,
			CreatedAt:  info.CreatedAt,
		})
	}

	return infos, total, nil
}

func GetAppDetails(ctx context.Context, appID string, offset, limit int32) ([]*npool.Detail, uint32, error) {
	conds := &ledgermgrdetailpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
	}
	details, total, err := ledgermgrdetailcli.GetDetails(ctx, conds, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(details) == 0 {
		return nil, 0, nil
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
			Op:    cruder.IN,
			Value: coinTypeIDs,
		},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*appcoinpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.CoinTypeID] = coin
	}

	userIDs := []string{}
	for _, info := range details {
		userIDs = append(userIDs, info.UserID)
	}

	users, _, err := usermwcli.GetManyUsers(ctx, userIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("fail get users: %v", err)
	}

	userMap := map[string]*appusermwpb.User{}
	for _, user := range users {
		userMap[user.ID] = user
	}

	infos := []*npool.Detail{}
	for _, detail := range details {
		user, ok := userMap[detail.UserID]
		if !ok {
			continue
		}
		coin, ok := coinMap[detail.CoinTypeID]
		if !ok {
			continue
		}
		infos = append(infos, &npool.Detail{
			CoinTypeID:   detail.CoinTypeID,
			CoinName:     coin.Name,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			IOType:       detail.IOType,
			IOSubType:    detail.IOSubType,
			Amount:       detail.Amount,
			IOExtra:      detail.IOExtra,
			CreatedAt:    detail.CreatedAt,
			UserID:       detail.UserID,
			PhoneNO:      user.PhoneNO,
			EmailAddress: user.EmailAddress,
		})
	}

	return infos, total, nil
}
