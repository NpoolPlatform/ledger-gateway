package profit

import (
	"context"
	"encoding/json"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	constant "github.com/NpoolPlatform/ledger-gateway/pkg/const"
	statementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	types "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	statementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
)

type baseHandler struct {
	*Handler
	statements  map[string]map[string]map[string][]*statementmwpb.Statement // AppGoodID -> CoinTypeID -> OrderID
	total       uint32
	orders      map[string]*ordermwpb.Order
	appCoins    map[string]*appcoinmwpb.Coin
	appGoods    map[string]*appgoodmwpb.Good
	ioType      types.IOType
	ioSubTypes  []types.IOSubType
	coinTypeIDs []string
}

func (h *baseHandler) getStatements(ctx context.Context) error {
	_ioSubTypes := []uint32{}
	for _, subType := range h.ioSubTypes {
		_ioSubTypes = append(_ioSubTypes, uint32(subType))
	}

	offset := int32(0)
	limit := constant.DefaultRowLimit
	for {
		statements, _, err := statementcli.GetStatements(ctx, &statementmwpb.Conds{
			AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			IOType:      &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(h.ioType)},
			IOSubTypes:  &basetypes.Uint32SliceVal{Op: cruder.IN, Value: _ioSubTypes},
			StartAt:     &basetypes.Uint32Val{Op: cruder.EQ, Value: h.StartAt},
			EndAt:       &basetypes.Uint32Val{Op: cruder.EQ, Value: h.EndAt},
			CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.coinTypeIDs},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(statements) == 0 {
			break
		}
		for _, statement := range statements {
			e := struct {
				OrderID   string
				AppGoodID string
			}{}
			if err := json.Unmarshal([]byte(statement.IOExtra), &e); err != nil {
				logger.Sugar().Errorw(
					"getStatements",
					"IOExtra", statement.IOExtra,
					"Error", err,
				)
				continue
			}
			order, ok := h.orders[e.OrderID]
			if !ok {
				logger.Sugar().Errorw(
					"getStatements",
					"OrderID", e.OrderID,
					"Error", "Invalid order",
				)
				continue
			}
			if order.AppGoodID != e.AppGoodID {
				logger.Sugar().Errorw(
					"getStatements",
					"OrderAppGoodID", order.AppGoodID,
					"OrderGoodID", order.GoodID,
					"StatementAppGoodID", e.AppGoodID,
					"Error", "Invalid statement",
				)
				continue
			}
			goodStatements, ok := h.statements[order.AppGoodID]
			if !ok {
				goodStatements = map[string]map[string][]*statementmwpb.Statement{}
			}
			coinStatements, ok := goodStatements[order.CoinTypeID]
			if !ok {
				coinStatements = map[string][]*statementmwpb.Statement{}
			}
			orderStatements, ok := coinStatements[order.ID]
			if !ok {
				orderStatements = []*statementmwpb.Statement{}
			}
			orderStatements = append(orderStatements, statement)
			coinStatements[order.ID] = orderStatements
			goodStatements[order.CoinTypeID] = coinStatements
			h.statements[order.AppGoodID] = goodStatements
		}
		offset += limit
	}
	return nil
}

func (h *baseHandler) getOrders(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit
	infos := []*ordermwpb.Order{}
	for {
		orders, _, err := ordermwcli.GetOrders(ctx, &ordermwpb.Conds{
			AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			break
		}

		infos = append(infos, orders...)
		offset += limit
	}
	for _, order := range infos {
		h.orders[order.ID] = order
	}
	return nil
}

func (h *baseHandler) getAppCoins(ctx context.Context) error {
	for _, val := range h.appGoods {
		h.coinTypeIDs = append(h.coinTypeIDs, val.CoinTypeID)
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.coinTypeIDs},
	}, 0, int32(len(h.coinTypeIDs)))
	if err != nil {
		return err
	}
	for _, coin := range coins {
		h.appCoins[coin.CoinTypeID] = coin
	}
	return nil
}

func (h *baseHandler) getAppGoods(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit
	for {
		goods, total, err := appgoodmwcli.GetGoods(ctx, &appgoodmwpb.Conds{
			AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(goods) == 0 {
			break
		}

		h.total = total
		for _, good := range goods {
			h.appGoods[good.ID] = good
		}
		offset += limit
	}
	return nil
}
