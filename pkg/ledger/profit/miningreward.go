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
	"github.com/shopspring/decimal"
)

type rewardHandler struct {
	*Handler
	statements []*statementmwpb.Statement
	appCoins   map[string]*appcoinmwpb.Coin
	orders     map[string]*ordermwpb.Order
	infos      []*npool.MiningReward
	total      uint32
}

func (h *rewardHandler) getAppCoins(ctx context.Context) error {
	ids := []string{}
	for _, val := range h.statements {
		ids = append(ids, val.CoinTypeID)
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: ids},
	}, 0, int32(len(ids)))
	if err != nil {
		return err
	}
	for _, coin := range coins {
		h.appCoins[coin.CoinTypeID] = coin
	}
	return nil
}

//nolint
func (h *rewardHandler) getOrders(ctx context.Context) error {
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

func (h *rewardHandler) getStatements(ctx context.Context) error {
	conds := &statementmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		IOType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(types.IOType_Incoming)},
	}
	if h.StartAt != nil {
		conds.StartAt = &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.StartAt}
	}
	if h.EndAt != nil {
		conds.EndAt = &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.EndAt}
	}
	if h.SimulateOnly != nil && *h.SimulateOnly {
		conds.IOSubType = &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(types.IOSubType_SimulateMiningBenefit)}
	} else {
		conds.IOSubTypes = &basetypes.Uint32SliceVal{Op: cruder.IN, Value: []uint32{
			uint32(types.IOSubType_MiningBenefit),
			uint32(types.IOSubType_SimulateMiningBenefit),
		}}
	}
	statements, total, err := statementcli.GetStatements(ctx, conds, h.Offset, h.Limit)
	if err != nil {
		return err
	}
	h.statements = statements
	h.total = total
	return nil
}

func (h *rewardHandler) formalize() {
	for _, statement := range h.statements {
		coin, ok := h.appCoins[statement.CoinTypeID]
		if !ok {
			continue
		}
		e := struct {
			OrderID   string
			AppGoodID string
		}{}
		if err := json.Unmarshal([]byte(statement.IOExtra), &e); err != nil {
			continue
		}
		order, ok := h.orders[e.OrderID]
		if !ok {
			continue
		}
		if order.AppGoodID != e.AppGoodID {
			continue
		}

		rewardAmount, err := decimal.NewFromString(statement.Amount)
		if err != nil {
			break
		}
		units, err := decimal.NewFromString(order.Units)
		if err != nil {
			continue
		}

		h.infos = append(h.infos, &npool.MiningReward{
			ID:                  statement.ID,
			EntID:               statement.EntID,
			AppID:               statement.AppID,
			UserID:              statement.UserID,
			CoinTypeID:          statement.CoinTypeID,
			CoinName:            coin.Name,
			CoinLogo:            coin.Logo,
			CoinUnit:            coin.Unit,
			IOType:              statement.IOType,
			IOSubType:           statement.IOSubType,
			RewardAmount:        statement.Amount,
			RewardAmountPerUnit: rewardAmount.Div(units).String(),
			Units:               order.Units,
			Extra:               statement.IOExtra,
			AppGoodID:           e.AppGoodID,
			OrderID:             e.OrderID,
			CreatedAt:           statement.CreatedAt,
		})
	}
}

func (h *Handler) GetMiningRewards(ctx context.Context) ([]*npool.MiningReward, uint32, error) {
	handler := &rewardHandler{
		Handler:    h,
		appCoins:   map[string]*appcoinmwpb.Coin{},
		orders:     map[string]*ordermwpb.Order{},
		statements: []*statementmwpb.Statement{},
	}
	if err := handler.getStatements(ctx); err != nil {
		return nil, 0, err
	}
	if len(handler.statements) == 0 {
		return nil, handler.total, nil
	}
	if err := handler.getOrders(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()
	return handler.infos, handler.total, nil
}
