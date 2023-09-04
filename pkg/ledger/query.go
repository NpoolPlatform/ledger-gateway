package ledger

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appusermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

type queryHandler struct {
	*Handler
	ledgers  map[string]*ledgermwpb.Ledger
	appcoins []*appcoinmwpb.Coin
	appusers map[string]*appusermwpb.User
	infos    []*npool.Ledger
	total    uint32
}

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

func (h *queryHandler) getAppCoins(ctx context.Context) error {
	coins, total, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
	}, h.Offset, h.Limit)
	if err != nil {
		return err
	}
	h.appcoins = coins
	h.total = total
	return nil
}

func (h *queryHandler) getLedgers(ctx context.Context) error {
	infos, _, err := ledgermwcli.GetLedgers(ctx, h.setConds(), 0, int32(len(h.appcoins)))
	if err != nil {
		return err
	}
	for _, val := range infos {
		h.ledgers[val.CoinTypeID] = val
	}
	return nil
}

func (h *queryHandler) getAppUsers(ctx context.Context) error {
	userIDs := []string{}
	for _, info := range h.ledgers {
		userIDs = append(userIDs, info.UserID)
	}

	users, _, err := usermwcli.GetUsers(ctx, &appusermwpb.Conds{
		IDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: userIDs},
	}, 0, int32(len(userIDs)))
	if err != nil {
		return err
	}

	for _, user := range users {
		h.appusers[user.ID] = user
	}
	return nil
}

func (h *queryHandler) formalize() {
	for _, coin := range h.appcoins {
		ledger, ok := h.ledgers[coin.CoinTypeID]
		if ok {
			h.infos = append(h.infos, &npool.Ledger{
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
			h.infos = append(h.infos, &npool.Ledger{
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
}

func (h *Handler) GetLedgers(ctx context.Context) ([]*npool.Ledger, uint32, error) {
	handler := &queryHandler{
		Handler:  h,
		appcoins: []*appcoinmwpb.Coin{},
		appusers: map[string]*appusermwpb.User{},
	}

	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getLedgers(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()
	return handler.infos, handler.total, nil
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

	users, _, err := usermwcli.GetUsers(ctx, &appusermwpb.Conds{
		IDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: userIDs},
	}, 0, int32(len(userIDs)))
	if err != nil {
		return nil, 0, fmt.Errorf("fail get users: %v", err)
	}

	userMap := map[string]*appusermwpb.User{}
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
