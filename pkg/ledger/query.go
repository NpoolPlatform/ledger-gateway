package ledger

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

func (h *Handler) setConds() *ledgermwpb.Conds {
	conds := &ledgermwpb.Conds{}
	if h.AppID != nil {
		conds.AppID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID}
	}
	if h.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}
	if h.CoinTypeIDs != nil {
		conds.CoinTypeIDs = &basetypes.StringSliceVal{Op: cruder.IN, Value: h.CoinTypeIDs}
	}

	return conds
}

func (h *Handler) GetLedgers(ctx context.Context) ([]*npool.Ledger, uint32, error) {
	coins, total, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
	}, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}
	if len(coins) == 0 {
		return nil, 0, nil
	}

	for _, coin := range coins {
		h.CoinTypeIDs = append(h.CoinTypeIDs, coin.CoinTypeID)
	}

	infos, _, err := ledgermwcli.GetLedgers(ctx, h.setConds(), 0, int32(len(coins)))
	if err != nil {
		return nil, 0, err
	}

	ledgerMap := map[string]*ledgermwpb.Ledger{}
	for _, ledger := range infos {
		ledgerMap[ledger.CoinTypeID] = ledger
	}

	ledgers := []*npool.Ledger{}
	for _, coin := range coins {
		ledger, ok := ledgerMap[coin.CoinTypeID]
		if ok {
			ledgers = append(ledgers, &npool.Ledger{
				CoinTypeID:   coin.CoinTypeID,
				CoinName:     coin.Name,
				DisplayNames: coin.DisplayNames,
				CoinLogo:     coin.Logo,
				CoinUnit:     coin.Unit,
				CoinDisabled: coin.Disabled,
				CoinDisplay:  coin.Display,
				Incoming:     ledger.Incoming,
				Locked:       ledger.Locked,
				Outcoming:    ledger.Outcoming,
				Spendable:    ledger.Spendable,
			})
		} else {
			ledgers = append(ledgers, &npool.Ledger{
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

	return ledgers, total, nil
}

func (h *Handler) GetAppLedgers(ctx context.Context) ([]*npool.Ledger, uint32, error) {
	infos, total, err := ledgermwcli.GetLedgers(ctx, h.setConds(), h.Offset, h.Limit)
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
			Value: *h.AppID,
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

	ledgers := []*npool.Ledger{}
	for _, ledger := range infos {
		user, ok := userMap[ledger.UserID]
		if !ok {
			continue
		}
		coin, ok := coinMap[ledger.CoinTypeID]
		if !ok {
			continue
		}
		ledgers = append(ledgers, &npool.Ledger{
			CoinTypeID:   coin.CoinTypeID,
			CoinName:     coin.Name,
			DisplayNames: coin.DisplayNames,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			Incoming:     ledger.Incoming,
			Locked:       ledger.Locked,
			Outcoming:    ledger.Outcoming,
			Spendable:    ledger.Spendable,
			UserID:       ledger.UserID,
			PhoneNO:      user.PhoneNO,
			EmailAddress: user.EmailAddress,
		})
	}
	return ledgers, total, nil
}
