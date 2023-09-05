package profit

import (
	"context"

	types "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	"github.com/shopspring/decimal"
)

type profitHandler struct {
	BaseHandler
	profits []*npool.Profit
}

func (h *profitHandler) formalize() {
	infos := map[string]*npool.Profit{}
	for _, val := range h.statements {
		coin, ok := h.appCoins[val.CoinTypeID]
		if !ok {
			continue
		}

		p, ok := infos[val.CoinTypeID]
		if !ok {
			p = &npool.Profit{
				CoinTypeID:   val.CoinTypeID,
				CoinName:     coin.Name,
				DisplayNames: coin.DisplayNames,
				CoinLogo:     coin.Logo,
				CoinUnit:     coin.Unit,
				Incoming:     decimal.NewFromInt(0).String(),
			}
		}

		p.Incoming = decimal.RequireFromString(p.Incoming).
			Add(decimal.RequireFromString(val.Amount)).
			String()

		infos[val.CoinTypeID] = p
	}

	for _, info := range infos {
		h.profits = append(h.profits, info)
	}
}

func (h *Handler) GetIntervalProfits(ctx context.Context) ([]*npool.Profit, uint32, error) {
	handler := &profitHandler{
		BaseHandler: BaseHandler{
			Handler:  h,
			appCoins: map[string]*appcoinmwpb.Coin{},
		},
	}
	if err := handler.getStatements(ctx, types.IOType_Incoming, []types.IOSubType{types.IOSubType_MiningBenefit}); err != nil {
		return nil, 0, err
	}
	if len(handler.statements) == 0 {
		return nil, 0, nil
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()
	return handler.profits, handler.total, nil
}
