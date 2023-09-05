package profit

import (
	"context"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	constant "github.com/NpoolPlatform/ledger-gateway/pkg/const"
	statementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	types "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	goodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"
	statementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
)

type BaseHandler struct {
	*Handler
	statements []*statementmwpb.Statement
	total      uint32
	orders     map[string]*ordermwpb.Order
	appCoins   map[string]*appcoinmwpb.Coin
	goods      map[string]*goodmwpb.Good
}

func (h *BaseHandler) getStatements(ctx context.Context, ioType types.IOType, ioSubTypes []types.IOSubType) error {
	_ioSubTypes := []uint32{}
	for _, subType := range ioSubTypes {
		_ioSubTypes = append(_ioSubTypes, uint32(subType))
	}

	statements := []*statementmwpb.Statement{}
	offset := int32(0)
	limit := constant.DefaultRowLimit
	for {
		sts, _total, err := statementcli.GetStatements(ctx, &statementmwpb.Conds{
			AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			IOType:     &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ioType)},
			IOSubTypes: &basetypes.Uint32SliceVal{Op: cruder.IN, Value: _ioSubTypes},
			StartAt:    &basetypes.Uint32Val{Op: cruder.EQ, Value: h.StartAt},
			EndAt:      &basetypes.Uint32Val{Op: cruder.EQ, Value: h.EndAt},
		}, offset, limit)
		if err != nil {
			return err
		}
		h.total = _total
		if len(sts) == 0 {
			break
		}
		statements = append(statements, sts...)
		offset += limit
	}
	h.statements = statements
	return nil
}

func (h *BaseHandler) getOrders(ctx context.Context) error {
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

func (h *BaseHandler) getAppCoins(ctx context.Context) error {
	coinTypeIDs := []string{}
	for _, val := range h.statements {
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}
	if h.goods != nil {
		for _, val := range h.goods {
			coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
		}
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return err
	}
	for _, coin := range coins {
		h.appCoins[coin.CoinTypeID] = coin
	}
	return nil
}

func (h *BaseHandler) getGoods(ctx context.Context) error {
	ids := []string{}
	for _, order := range h.orders {
		ids = append(ids, order.GoodID)
	}

	goods, _, err := goodmwcli.GetGoods(ctx, &goodmwpb.Conds{
		IDs: &basetypes.StringSliceVal{
			Op:    cruder.IN,
			Value: ids,
		},
	}, 0, int32(len(ids)))
	if err != nil {
		return err
	}

	for _, good := range goods {
		h.goods[good.ID] = good
	}
	return nil
}
