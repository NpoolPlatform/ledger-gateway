package ledger

import (
	"context"
	"fmt"

	commonpb "github.com/NpoolPlatform/message/npool"
	appusermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	ledgermgrdetailpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/detail"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	ledgermgrdetailcli "github.com/NpoolPlatform/ledger-manager/pkg/client/detail"
	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"
	coininfopb "github.com/NpoolPlatform/message/npool/coininfo"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
)

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

	coins, err := coininfocli.GetCoinInfos(ctx, cruder.NewFilterConds())
	if err != nil {
		return nil, 0, err
	}

	coinMap := map[string]*coininfopb.CoinInfo{}
	for _, coin := range coins {
		coinMap[coin.ID] = coin
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
			total -= 1
			continue
		}
		coin, ok := coinMap[detail.CoinTypeID]
		if !ok {
			total -= 1
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
			PhoneNo:      user.PhoneNO,
			EmailAddress: user.EmailAddress,
		})
	}

	return infos, total, nil
}
