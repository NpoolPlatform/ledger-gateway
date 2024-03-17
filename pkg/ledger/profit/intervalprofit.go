package profit

import (
	"context"
	"encoding/json"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	constant "github.com/NpoolPlatform/ledger-gateway/pkg/const"
	statementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	types "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/profit"
	statementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type profitHandler struct {
	*Handler
	infos      []*npool.Profit
	statements map[string][]*statementmwpb.Statement
	appCoins   []*appcoinmwpb.Coin
	orders     map[string]*ordermwpb.Order
	total      uint32
}

func (h *profitHandler) getStatements(ctx context.Context) error {
	ids := []string{}
	for _, coin := range h.appCoins {
		if _, err := uuid.Parse(coin.CoinTypeID); err != nil {
			continue
		}
		ids = append(ids, coin.CoinTypeID)
	}
	offset := int32(0)
	limit := constant.DefaultRowLimit
	for {
		statements, _, err := statementcli.GetStatements(ctx, &statementmwpb.Conds{
			AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			IOType:      &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(types.IOType_Incoming)},
			IOSubType:   &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(types.IOSubType_MiningBenefit)},
			StartAt:     &basetypes.Uint32Val{Op: cruder.EQ, Value: h.StartAt},
			EndAt:       &basetypes.Uint32Val{Op: cruder.EQ, Value: h.EndAt},
			CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: ids},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(statements) == 0 {
			break
		}

		for _, statement := range statements {
			statements, ok := h.statements[statement.CoinTypeID]
			if !ok {
				h.statements[statement.CoinTypeID] = []*statementmwpb.Statement{}
			}
			statements = append(statements, statement)
			h.statements[statement.CoinTypeID] = statements
		}
		offset += limit
	}
	return nil
}

//nolint
func (h *profitHandler) getOrders(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit
	for {
		orders, _, err := ordermwcli.GetOrders(ctx, &ordermwpb.Conds{
			AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			OrderStates: &basetypes.Uint32SliceVal{Op: cruder.NIN, Value: []uint32{
				uint32(ordertypes.OrderState_OrderStateCreated),
				uint32(ordertypes.OrderState_OrderStateWaitPayment),
				uint32(ordertypes.OrderState_OrderStatePaymentTimeout),
				uint32(ordertypes.OrderState_OrderStatePreCancel),
				uint32(ordertypes.OrderState_OrderStateRestoreCanceledStock),
				uint32(ordertypes.OrderState_OrderStateCancelAchievement),
				uint32(ordertypes.OrderState_OrderStateDeductLockedCommission),
				uint32(ordertypes.OrderState_OrderStateReturnCanceledBalance),
				uint32(ordertypes.OrderState_OrderStateCanceledTransferBookKeeping),
				uint32(ordertypes.OrderState_OrderStateCancelUnlockPaymentAccount),
				uint32(ordertypes.OrderState_OrderStateUpdateCanceledChilds),
				uint32(ordertypes.OrderState_OrderStateCanceled),
			}}}, offset, limit)
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			break
		}

		for _, order := range orders {
			h.orders[order.EntID] = order
		}
		offset += limit
	}
	return nil
}

func (h *profitHandler) getAppCoins(ctx context.Context) error {
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

func (h *profitHandler) formalize() {
	profits := map[string]*npool.Profit{}
	for _, coin := range h.appCoins {
		p, ok := profits[coin.CoinTypeID]
		if !ok {
			p = &npool.Profit{
				AppID:        coin.AppID,
				UserID:       *h.UserID,
				CoinTypeID:   coin.CoinTypeID,
				CoinName:     coin.Name,
				DisplayNames: coin.DisplayNames,
				CoinLogo:     coin.Logo,
				CoinUnit:     coin.Unit,
				Incoming:     decimal.NewFromInt(0).String(),
			}
		}

		statements, ok := h.statements[coin.CoinTypeID]
		if ok {
			for _, statement := range statements {
				e := struct {
					OrderID   string
					AppGoodID string
				}{}
				if err := json.Unmarshal([]byte(statement.IOExtra), &e); err != nil {
					continue
				}
				_, ok := h.orders[e.OrderID]
				if ok {
					p.Incoming = decimal.RequireFromString(p.Incoming).
						Add(decimal.RequireFromString(statement.Amount)).
						String()
					p.AppID = statement.AppID
					p.UserID = statement.UserID
				}
			}
		}
		profits[coin.CoinTypeID] = p
	}
	for _, val := range profits {
		h.infos = append(h.infos, val)
	}
}

func (h *Handler) GetIntervalProfits(ctx context.Context) ([]*npool.Profit, uint32, error) {
	handler := &profitHandler{
		Handler:    h,
		appCoins:   []*appcoinmwpb.Coin{},
		orders:     map[string]*ordermwpb.Order{},
		statements: map[string][]*statementmwpb.Statement{},
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	if len(handler.appCoins) == 0 {
		return nil, 0, nil
	}
	if err := handler.getOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getStatements(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()
	return handler.infos, handler.total, nil
}
