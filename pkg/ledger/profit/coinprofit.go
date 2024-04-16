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

	"github.com/shopspring/decimal"
)

type coinProfitHandler struct {
	*Handler
	infos    []*npool.CoinProfit
	appCoins []*appcoinmwpb.Coin
	profits  map[string]*profitmwpb.Profit
	total    uint32
}

func (h *coinProfitHandler) getAppCoins(ctx context.Context) error {
	coins, total, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}, h.Offset, h.Limit)
	if err != nil {
		return err
	}
	h.total = total
	h.appCoins = coins
	return nil
}

func (h *coinProfitHandler) getProfits(ctx context.Context) error {
	coinTypeIDs := func() (_coinTypeIDs []string) {
		for _, appCoin := range h.appCoins {
			_coinTypeIDs = append(_coinTypeIDs, appCoin.CoinTypeID)
		}
		return
	}()
	profits, _, err := profitmwcli.GetProfits(ctx, &profitmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return err
	}
	for _, profit := range profits {
		h.profits[profit.CoinTypeID] = profit
	}
	return nil
}

func (h *coinProfitHandler) formalize() {
	for _, coin := range h.appCoins {
		h.infos = append(h.infos, &npool.CoinProfit{
			AppID:        coin.AppID,
			UserID:       *h.UserID,
			CoinTypeID:   coin.CoinTypeID,
			CoinName:     coin.Name,
			DisplayNames: coin.DisplayNames,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			Incoming: func() string {
				profit, ok := h.profits[coin.CoinTypeID]
				if !ok {
					return decimal.NewFromInt(0).String()
				}
				return profit.Incoming
			}(),
		})
	}
}

func (h *Handler) GetCoinProfits(ctx context.Context) ([]*npool.CoinProfit, uint32, error) {
	handler := &coinProfitHandler{
		Handler:  h,
		appCoins: []*appcoinmwpb.Coin{},
		profits:  map[string]*profitmwpb.Profit{},
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	if len(handler.appCoins) == 0 {
		return nil, 0, nil
	}
	if err := handler.getProfits(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()

	return handler.infos, handler.total, nil
}
