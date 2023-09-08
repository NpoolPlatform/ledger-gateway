package ledger

import (
	"context"

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
	ledgers      map[string]map[string]*ledgermwpb.Ledger
	appCoins     map[string]*appcoinmwpb.Coin
	appUsers     map[string]*appusermwpb.User
	infos        []*npool.Ledger
	totalLedgers uint32
	totalCoins   uint32
}

func (h *queryHandler) getAppCoins(ctx context.Context, conds *appcoinmwpb.Conds, offset, limit int32) error {
	coins, total, err := appcoinmwcli.GetCoins(ctx, conds, offset, limit)
	if err != nil {
		return err
	}
	for _, coin := range coins {
		h.appCoins[coin.CoinTypeID] = coin
	}
	h.totalCoins = total
	return nil
}

func (h *queryHandler) getLedgers(ctx context.Context, conds *ledgermwpb.Conds, offset, limit int32) error {
	ledgers, total, err := ledgermwcli.GetLedgers(ctx, conds, offset, limit)
	if err != nil {
		return err
	}
	for _, ledger := range ledgers {
		ledgers, ok := h.ledgers[ledger.UserID]
		if !ok {
			ledgers = map[string]*ledgermwpb.Ledger{}
		}
		ledgers[ledger.CoinTypeID] = ledger
		h.ledgers[ledger.UserID] = ledgers
	}
	h.totalLedgers = total
	return nil
}

func (h *queryHandler) getAppUsers(ctx context.Context) error {
	userIDs := []string{}
	for userID := range h.ledgers {
		userIDs = append(userIDs, userID)
	}
	users, _, err := usermwcli.GetUsers(ctx, &appusermwpb.Conds{
		IDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: userIDs},
	}, 0, int32(len(userIDs)))
	if err != nil {
		return err
	}
	for _, user := range users {
		h.appUsers[user.ID] = user
	}
	return nil
}

func (h *queryHandler) prepareAppLedgers(ctx context.Context) error {
	// Get offset/limit ledgers
	if err := h.getLedgers(ctx, &ledgermwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}, h.Offset, h.Limit); err != nil {
		return err
	}
	coinTypeIDs := []string{}
	for _, ledgers := range h.ledgers {
		for coinTypeID := range ledgers {
			coinTypeIDs = append(coinTypeIDs, coinTypeID)
		}
	}
	conds := &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}
	// Get ledger coins
	if err := h.getAppCoins(ctx, conds, 0, int32(len(coinTypeIDs))); err != nil {
		return err
	}
	return nil
}

func (h *queryHandler) prepareUserLedgers(ctx context.Context) error {
	// Get offset/limit coins
	if err := h.getAppCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}, h.Offset, h.Limit); err != nil {
		return err
	}
	coinTypeIDs := []string{}
	for coinTypeID := range h.appCoins {
		coinTypeIDs = append(coinTypeIDs, coinTypeID)
	}
	// Get coin ledgers
	conds := &ledgermwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}
	if err := h.getLedgers(ctx, conds, 0, int32(len(coinTypeIDs))); err != nil {
		return err
	}
	return nil
}

func (h *queryHandler) formalize(ledger *ledgermwpb.Ledger, coin *appcoinmwpb.Coin, user *appusermwpb.User) {
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
		PhoneNO:      user.PhoneNO,
		EmailAddress: user.EmailAddress,
	})
}

func (h *queryHandler) formalizeAppLedgers() {
	for userID, ledgers := range h.ledgers {
		user, ok := h.appUsers[userID]
		if !ok {
			continue
		}
		for coinTypeID, ledger := range ledgers {
			coin, ok := h.appCoins[coinTypeID]
			if !ok {
				continue
			}
			h.formalize(ledger, coin, user)
		}
	}
}

func (h *queryHandler) formalizeUserLedgers() {
	user, ok := h.appUsers[*h.UserID]
	if !ok {
		return
	}

	ledgers, _ := h.ledgers[*h.UserID] //nolint
	for coinTypeID, coin := range h.appCoins {
		if ledgers != nil {
			ledger, ok := ledgers[coinTypeID]
			if ok {
				h.formalize(ledger, coin, user)
				continue
			}
		}
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
			PhoneNO:      user.PhoneNO,
			EmailAddress: user.EmailAddress,
		})
	}
}

func (h *Handler) GetLedgers(ctx context.Context) ([]*npool.Ledger, uint32, error) {
	handler := &queryHandler{
		Handler:  h,
		ledgers:  map[string]map[string]*ledgermwpb.Ledger{},
		appCoins: map[string]*appcoinmwpb.Coin{},
		appUsers: map[string]*appusermwpb.User{},
	}
	if h.UserID == nil {
		if err := handler.prepareAppLedgers(ctx); err != nil {
			return nil, 0, err
		}
	} else {
		if err := handler.prepareUserLedgers(ctx); err != nil {
			return nil, 0, err
		}
	}
	if err := handler.getAppUsers(ctx); err != nil {
		return nil, 0, err
	}

	if h.UserID == nil {
		handler.formalizeAppLedgers()
		return handler.infos, handler.totalLedgers, nil
	}

	handler.formalizeUserLedgers()
	return handler.infos, handler.totalCoins, nil
}
