package ledger

import (
	"context"
	"encoding/json"
	"fmt"

	orderstatemgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	"github.com/shopspring/decimal"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	commonpb "github.com/NpoolPlatform/message/npool"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	ledgermgrdetailpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/statement"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	ledgermgrdetailcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/statement"
	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	"github.com/google/uuid"
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
		if _, err := uuid.Parse(val.CoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: appID,
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

	infos := []*npool.Detail{}
	for _, info := range details {
		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			logger.Sugar().Warnw("GetDetails", "app coin not exist", "appID", appID, "user_id", userID)
			continue
		}

		infos = append(infos, &npool.Detail{
			CoinTypeID:   info.CoinTypeID,
			CoinName:     coin.Name,
			DisplayNames: coin.DisplayNames,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			IOType:       info.IOType,
			IOSubType:    info.IOSubType,
			Amount:       info.Amount,
			IOExtra:      info.IOExtra,
			CreatedAt:    info.CreatedAt,
		})
	}

	return infos, total, nil
}

// nolint
func GetMiningRewards(
	ctx context.Context,
	appID, userID string,
	start, end uint32,
	offset, limit int32) ([]*npool.MiningReward, uint32, error) {
	details, total, err := ledgermwcli.GetIntervalDetails(ctx, appID, userID, start, end, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(details) == 0 {
		return nil, total, nil
	}

	var coinTypeIDs []string
	for _, val := range details {
		if _, err := uuid.Parse(val.CoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: appID,
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

	ofs := int32(0)
	lim := int32(100)
	var orders []*ordermwpb.Order

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
	var length uint32

	for _, info := range details {
		if info.IOSubType != ledgermgrdetailpb.IOSubType_MiningBenefit {
			continue
		}
		if info.IOType != ledgermgrdetailpb.IOType_Incoming {
			continue
		}

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

		length += 1

		coin, ok := coinMap[info.CoinTypeID]
		if !ok {
			logger.Sugar().Warnw("GetMiningRewards", "app coin not exist", "appID", appID, "coin_type_id", info.CoinTypeID)
			continue
		}

		rewardAmount, err := decimal.NewFromString(info.Amount)
		if err != nil {
			logger.Sugar().Warnw("GetMiningRewards", "invalid amount", info.Amount)
			return nil, 0, err
		}

		units, err := decimal.NewFromString(order.Units)
		if err != nil {
			logger.Sugar().Warnw("GetMiningRewards", "invalid Units", order.Units)
			return nil, 0, err
		}

		infos = append(infos, &npool.MiningReward{
			CoinTypeID:          info.CoinTypeID,
			CoinName:            coin.Name,
			CoinLogo:            coin.Logo,
			CoinUnit:            coin.Unit,
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

	return infos, length, nil
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
		if _, err := uuid.Parse(val.CoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: appID,
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

	userIDs := []string{}
	for _, info := range details {
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
			DisplayNames: coin.DisplayNames,
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
