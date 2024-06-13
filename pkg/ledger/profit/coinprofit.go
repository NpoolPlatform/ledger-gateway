package profit

import (
	"context"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	constant "github.com/NpoolPlatform/ledger-gateway/pkg/const"
	profitmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/profit"
	statementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	ledgertypes "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	profitmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/profit"
	statementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"

	"github.com/shopspring/decimal"
)

type coinProfitHandler struct {
	*Handler
	infos       []*npool.CoinProfit
	appCoins    []*appcoinmwpb.Coin
	profits     map[string]*profitmwpb.Profit
	coinTypeIDs []string
	total       uint32
}

//nolint:dupl
func (h *coinProfitHandler) getStatements(ctx context.Context) error {
	if h.StartAt == nil && h.EndAt == nil {
		return nil
	}
	offset := int32(0)
	limit := constant.DefaultRowLimit
	conds := &statementmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		IOType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ledgertypes.IOType_Incoming)},
		IOSubTypes: &basetypes.Uint32SliceVal{Op: cruder.IN, Value: []uint32{
			uint32(ledgertypes.IOSubType_MiningBenefit),
			uint32(ledgertypes.IOSubType_SimulateMiningBenefit),
		}},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.coinTypeIDs},
	}
	if h.StartAt != nil {
		conds.StartAt = &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.StartAt}
	}
	if h.EndAt != nil {
		conds.EndAt = &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.EndAt}
	}

	for {
		statements, _, err := statementcli.GetStatements(ctx, conds, offset, limit)
		if err != nil {
			return err
		}
		if len(statements) == 0 {
			break
		}
		for _, statement := range statements {
			profit, ok := h.profits[statement.CoinTypeID]
			if ok {
				incoming, err := decimal.NewFromString(profit.Incoming)
				if err != nil {
					return err
				}
				amount, err := decimal.NewFromString(statement.Amount)
				if err != nil {
					return err
				}
				profit.Incoming = incoming.Add(amount).String()
				h.profits[statement.CoinTypeID] = profit
				continue
			}

			newProfit := &profitmwpb.Profit{
				CoinTypeID: statement.CoinTypeID,
				Incoming:   statement.Amount,
			}
			h.profits[statement.CoinTypeID] = newProfit
		}
		offset += limit
	}
	return nil
}

func (h *coinProfitHandler) getAppCoins(ctx context.Context) error {
	coins, total, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}, h.Offset, h.Limit)
	if err != nil {
		return err
	}

	for _, coin := range coins {
		h.coinTypeIDs = append(h.coinTypeIDs, coin.CoinTypeID)
	}

	h.total = total
	h.appCoins = coins
	return nil
}

func (h *coinProfitHandler) getProfits(ctx context.Context) error {
	if h.StartAt != nil || h.EndAt != nil {
		return nil
	}
	coinTypeIDs := func() (_coinTypeIDs []string) {
		for _, appCoin := range h.appCoins {
			_coinTypeIDs = append(_coinTypeIDs, appCoin.CoinTypeID)
		}
		return
	}()
	conds := &profitmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}

	profits, _, err := profitmwcli.GetProfits(ctx, conds, 0, int32(len(coinTypeIDs)))
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
	if err := h.CheckStartEndAt(); err != nil {
		return nil, 0, err
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
	if err := handler.getStatements(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()

	return handler.infos, handler.total, nil
}
