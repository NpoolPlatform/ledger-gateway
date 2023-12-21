package coupon

import (
	"context"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	couponwithdrawmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/withdraw/coupon"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	reviewtypes "github.com/NpoolPlatform/message/npool/basetypes/review/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw/coupon"
	couponwithdrawmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/withdraw/coupon"
	"github.com/NpoolPlatform/message/npool/review/mw/v2/review"
	reviewmwcli "github.com/NpoolPlatform/review-middleware/pkg/client/review"
)

type queryHandler struct {
	*Handler
	couponwithdraws []*couponwithdrawmwpb.CouponWithdraw
	appCoins        map[string]*appcoinmwpb.Coin
	reviewMessages  map[string]string
	infos           []*npool.CouponWithdraw
}

func (h *queryHandler) getAppCoins(ctx context.Context) error {
	ids := []string{}
	for _, withdraw := range h.couponwithdraws {
		ids = append(ids, withdraw.CoinTypeID)
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

func (h *queryHandler) getReviews(ctx context.Context) error {
	ids := []string{}
	for _, withdraw := range h.couponwithdraws {
		ids = append(ids, withdraw.ReviewID)
	}

	reviews, _, err := reviewmwcli.GetReviews(ctx, &review.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		EntIDs:     &basetypes.StringSliceVal{Op: cruder.IN, Value: ids},
		ObjectType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(reviewtypes.ReviewObjectType_ObjectCouponRandomCash)},
	}, 0, int32(len(ids)))
	if err != nil {
		return err
	}
	for _, r := range reviews {
		if r.State == reviewtypes.ReviewState_Rejected {
			h.reviewMessages[r.ObjectID] = r.Message
		}
	}
	return nil
}

func (h *queryHandler) formalize() {
	for _, withdraw := range h.couponwithdraws {
		coin, ok := h.appCoins[withdraw.CoinTypeID]
		if !ok {
			continue
		}

		h.infos = append(h.infos, &npool.CouponWithdraw{
			ID:           withdraw.ID,
			EntID:        withdraw.EntID,
			AppID:        withdraw.AppID,
			UserID:       withdraw.UserID,
			CoinTypeID:   withdraw.CoinTypeID,
			CoinName:     coin.CoinName,
			DisplayNames: coin.DisplayNames,
			CoinLogo:     coin.Logo,
			CoinUnit:     coin.Unit,
			Amount:       withdraw.Amount,
			CreatedAt:    withdraw.CreatedAt,
			State:        withdraw.State,
			Message:      h.reviewMessages[withdraw.EntID],
		})
	}
}

func (h *Handler) GetCouponWithdraws(ctx context.Context) ([]*npool.CouponWithdraw, uint32, error) {
	conds := &couponwithdrawmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}
	if h.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}
	withdraws, total, err := couponwithdrawmwcli.GetCouponWithdraws(ctx, conds, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}
	if len(withdraws) == 0 {
		return nil, total, nil
	}

	handler := &queryHandler{
		Handler:         h,
		couponwithdraws: withdraws,
		appCoins:        map[string]*appcoinmwpb.Coin{},
		reviewMessages:  map[string]string{},
	}

	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getReviews(ctx); err != nil {
		return nil, 0, err
	}

	handler.formalize()
	return handler.infos, total, nil
}

func (h *Handler) GetCouponWithdraw(ctx context.Context) (*npool.CouponWithdraw, error) {
	withdraw, err := couponwithdrawmwcli.GetCouponWithdraw(ctx, *h.EntID)
	if err != nil {
		return nil, err
	}
	if withdraw == nil {
		return nil, nil
	}

	handler := &queryHandler{
		Handler:         h,
		couponwithdraws: []*couponwithdrawmwpb.CouponWithdraw{withdraw},
		appCoins:        map[string]*appcoinmwpb.Coin{},
		reviewMessages:  map[string]string{},
	}

	if err := handler.getAppCoins(ctx); err != nil {
		return nil, err
	}
	if err := handler.getReviews(ctx); err != nil {
		return nil, err
	}
	handler.formalize()
	if len(handler.infos) == 0 {
		return nil, nil
	}
	return handler.infos[0], nil
}
