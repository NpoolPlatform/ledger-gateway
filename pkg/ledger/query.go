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
	ledgers  []*ledgermwpb.Ledger
	appcoins []*appcoinmwpb.Coin
	appusers map[string]*appusermwpb.User
	infos    []*npool.Ledger
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
	ids := []string{}
	for _, info := range h.ledgers {
		ids = append(ids, info.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		IDs:   &basetypes.StringSliceVal{Op: cruder.IN, Value: ids},
	}, 0, int32(len(ids)))
	if err != nil {
		return err
	}
	h.appcoins = coins
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
	ledgerMap := map[string]*ledgermwpb.Ledger{}
	for _, val := range h.ledgers {
		ledgerMap[val.CoinTypeID] = val
	}

	for _, coin := range h.appcoins {
		ledger, ok := ledgerMap[coin.CoinTypeID]
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
	coins, total, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}

	for _, coin := range coins {
		h.CoinTypeIDs = append(h.CoinTypeIDs, coin.CoinTypeID)
	}

	ledgers, _, err := ledgermwcli.GetLedgers(ctx, h.setConds(), 0, int32(len(coins)))
	if err != nil {
		return nil, 0, err
	}

	handler := &queryHandler{
		Handler:  h,
		appcoins: coins,
		ledgers:  ledgers,
	}

	handler.formalize()
	return handler.infos, total, nil
}

func (h *queryHandler) formalize1() {
	coinMap := map[string]*appcoinmwpb.Coin{}
	for _, coin := range h.appcoins {
		coinMap[coin.CoinTypeID] = coin
	}

	for _, val := range h.ledgers {
		user, ok := h.appusers[val.UserID]
		if !ok {
			continue
		}
		coin, ok := coinMap[val.CoinTypeID]
		if !ok {
			continue
		}
		h.infos = append(h.infos, &npool.Ledger{
			CoinTypeID:   coin.CoinTypeID,
			CoinName:     coin.Name,
			DisplayNames: coin.DisplayNames,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			Incoming:     val.Incoming,
			Locked:       val.Locked,
			Outcoming:    val.Outcoming,
			Spendable:    val.Spendable,
			UserID:       val.UserID,
			PhoneNO:      user.PhoneNO,
			EmailAddress: user.EmailAddress,
		})
	}
}

func (h *Handler) GetAppLedgers(ctx context.Context) ([]*npool.Ledger, uint32, error) {
	ledgers, total, err := ledgermwcli.GetLedgers(ctx, h.setConds(), h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}
	if len(ledgers) == 0 {
		return nil, 0, nil
	}
	handler := &queryHandler{
		Handler: h,
		ledgers: ledgers,
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, total, err
	}
	if err := handler.getAppUsers(ctx); err != nil {
		return nil, total, err
	}

	handler.formalize1()
	return handler.infos, total, nil
}
