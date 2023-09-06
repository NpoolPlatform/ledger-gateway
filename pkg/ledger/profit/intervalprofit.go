package profit

import (
	"context"

	types "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
    ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
    statementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
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
			for _, statements := range coinStatements {
				for _, val := range statements {
					p.Incoming = decimal.RequireFromString(p.Incoming).
						Add(decimal.RequireFromString(val.Amount)).
						String()
					infos[coinTypeID] = p
				}
			}
		}
	}

	for _, val := range infos {
		h.profits = append(h.profits, val)
	}
}

func (h *Handler) GetIntervalProfits(ctx context.Context) ([]*npool.Profit, uint32, error) {
	handler := &profitHandler{
		baseHandler: &baseHandler{
			Handler:    h,
            appCoins:   map[string]*appcoinmwpb.Coin{},
			appGoods:   map[string]*appgoodmwpb.Good{},
			orders:     map[string]*ordermwpb.Order{},
			statements: map[string]map[string]map[string][]*statementmwpb.Statement{},
			ioType:     types.IOType_Incoming,
			ioSubTypes: []types.IOSubType{types.IOSubType_MiningBenefit},
		},
	}
	if err := handler.getOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppGoods(ctx); err != nil {
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
