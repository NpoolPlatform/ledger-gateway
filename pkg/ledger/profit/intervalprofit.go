package profit

import (
	"context"

	types "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	"github.com/shopspring/decimal"
)

type profitHandler struct {
	*baseHandler
	profits []*npool.Profit
}

func (h *profitHandler) formalize() {
	infos := map[string]*npool.Profit{}
	for appGoodID, goodStatements := range h.statements {
		_, ok := h.appGoods[appGoodID]
		if !ok {
			continue
		}
		for coinTypeID, coinStatements := range goodStatements {
			coin, ok := h.appCoins[coinTypeID]
			if !ok {
				continue
			}
			for _, statements := range coinStatements {
				for _, val := range statements {
					p, ok := infos[coinTypeID]
					if !ok {
						p = &npool.Profit{
							CoinTypeID:   coinTypeID,
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
					infos[coinTypeID] = p
				}
			}

		}
	}
}

func (h *Handler) GetIntervalProfits(ctx context.Context) ([]*npool.Profit, uint32, error) {
	handler := &profitHandler{
		baseHandler: &baseHandler{
			Handler:    h,
			appCoins:   map[string]*appcoinmwpb.Coin{},
			ioType:     types.IOType_Incoming,
			ioSubTypes: []types.IOSubType{types.IOSubType_MiningBenefit},
		},
	}
	if err := handler.getOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getStatements(ctx); err != nil {
		return nil, 0, err
	}
	if len(handler.statements) == 0 {
		return nil, 0, nil
	}

	handler.formalize()
	return handler.profits, handler.total, nil
}
