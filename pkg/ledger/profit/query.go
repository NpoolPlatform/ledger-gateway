package profit

import (
	"context"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"

	profitmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/profit"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"

	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	profitmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/profit"
)

type queryHandler struct {
	*Handler
	appcoins map[string]*appcoinmwpb.Coin
	profits  []*profitmwpb.Profit
	infos    []*npool.Profit
}

// nolint
func (h *queryHandler) getAppCoins(ctx context.Context) error {
	coinTypeIDs := []string{}
	for _, profit := range h.profits {
		coinTypeIDs = append(coinTypeIDs, profit.CoinTypeID)
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return err
	}
	for _, coin := range coins {
		h.appcoins[coin.CoinTypeID] = coin
	}
	return nil
}

func (h *queryHandler) formalize() {
	for _, info := range h.profits {
		coin, ok := h.appcoins[info.CoinTypeID]
		if !ok {
			continue
		}
		h.infos = append(h.infos, &npool.Profit{
			CoinTypeID:   info.CoinTypeID,
			CoinName:     coin.Name,
			DisplayNames: coin.DisplayNames,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			Incoming:     info.Incoming,
		})
	}
}

func (h *Handler) GetProfits(ctx context.Context) ([]*npool.Profit, uint32, error) {
	conds := &profitmwpb.Conds{}
	if h.AppID != nil {
		conds.AppID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID}
	}
	if h.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}
	profits, total, err := profitmwcli.GetProfits(ctx, conds, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}
	if len(profits) == 0 {
		return nil, total, nil
	}

	handler := &queryHandler{
		Handler:  h,
		profits:  profits,
		appcoins: map[string]*appcoinmwpb.Coin{},
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	handler.formalize()

	return handler.infos, total, nil
}
