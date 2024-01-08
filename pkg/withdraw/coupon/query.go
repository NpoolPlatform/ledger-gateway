package coupon

import (
	"context"

	appusermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	allocatedmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	couponwithdrawmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/withdraw/coupon"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	appusermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	reviewtypes "github.com/NpoolPlatform/message/npool/basetypes/review/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw/coupon"
	couponwithdrawmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/withdraw/coupon"
	reviewmwpb "github.com/NpoolPlatform/message/npool/review/mw/v2/review"
	reviewmwcli "github.com/NpoolPlatform/review-middleware/pkg/client/review"
)

type queryHandler struct {
	*Handler
	couponwithdraws []*couponwithdrawmwpb.CouponWithdraw
	appcoins        map[string]*appcoinmwpb.Coin
	appusers        map[string]*appusermwpb.User
	allocateds      map[string]*allocatedmwpb.Coupon
	reviews         map[string]*reviewmwpb.Review
	infos           []*npool.CouponWithdraw
}

func (h *queryHandler) getAppUsers(ctx context.Context) error {
	ids := []string{}
	for _, cw := range h.couponwithdraws {
		ids = append(ids, cw.UserID)
	}
	infos, _, err := appusermwcli.GetUsers(ctx, &appusermwpb.Conds{
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: ids},
	}, 0, int32(len(ids)))
	if err != nil {
		return err
	}
	for _, info := range infos {
		h.appusers[info.EntID] = info
	}
	return nil
}

func (h *queryHandler) getAllocateds(ctx context.Context) error {
	ids := []string{}
	for _, cw := range h.couponwithdraws {
		ids = append(ids, cw.AllocatedID)
	}
	allocateds, _, err := allocatedmwcli.GetCoupons(ctx, &allocatedmwpb.Conds{
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: ids},
	}, 0, int32(len(ids)))
	if err != nil {
		return err
	}
	for _, al := range allocateds {
		h.allocateds[al.EntID] = al
	}
	return nil
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
		h.appcoins[coin.CoinTypeID] = coin
	}
	return nil
}

func (h *queryHandler) getReviews(ctx context.Context) error {
	ids := []string{}
	for _, withdraw := range h.couponwithdraws {
		ids = append(ids, withdraw.ReviewID)
	}

	reviews, _, err := reviewmwcli.GetReviews(ctx, &reviewmwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		EntIDs:     &basetypes.StringSliceVal{Op: cruder.IN, Value: ids},
		ObjectType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(reviewtypes.ReviewObjectType_ObjectRandomCouponCash)},
	}, 0, int32(len(ids)))
	if err != nil {
		return err
	}
	for _, r := range reviews {
		h.reviews[r.EntID] = r
	}
	return nil
}

func (h *queryHandler) formalize() {
	for _, cw := range h.couponwithdraws {
		coin, ok := h.appcoins[cw.CoinTypeID]
		if !ok {
			continue
		}
		allocated, ok := h.allocateds[cw.AllocatedID]
		if !ok {
			continue
		}
		appuser, ok := h.appusers[cw.UserID]
		if !ok {
			continue
		}
		_review, ok := h.reviews[cw.ReviewID]
		if !ok {
			continue
		}

		message := ""
		if _review.State == reviewtypes.ReviewState_Rejected {
			message = _review.Message
		}

		h.infos = append(h.infos, &npool.CouponWithdraw{
			ID:            cw.ID,
			EntID:         cw.EntID,
			AppID:         cw.AppID,
			UserID:        cw.UserID,
			CoinTypeID:    cw.CoinTypeID,
			CoinName:      coin.CoinName,
			DisplayNames:  coin.DisplayNames,
			CoinLogo:      coin.Logo,
			CoinUnit:      coin.Unit,
			Amount:        cw.Amount,
			State:         cw.State,
			Message:       message,
			ReviewID:      cw.ReviewID,
			ReviewUintID:  _review.ID,
			AllocatedID:   cw.AllocatedID,
			CouponID:      allocated.CouponID,
			CouponName:    allocated.CouponName,
			CouponMessage: allocated.Message,
			PhoneNO:       appuser.PhoneNO,
			EmailAddress:  appuser.EmailAddress,
			CreatedAt:     cw.CreatedAt,
			UpdatedAt:     cw.UpdatedAt,
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
		appcoins:        map[string]*appcoinmwpb.Coin{},
		allocateds:      map[string]*allocatedmwpb.Coupon{},
		appusers:        map[string]*appusermwpb.User{},
		reviews:         map[string]*reviewmwpb.Review{},
	}

	if err := handler.getAppCoins(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getReviews(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAllocateds(ctx); err != nil {
		return nil, 0, err
	}
	if err := handler.getAppUsers(ctx); err != nil {
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
		appcoins:        map[string]*appcoinmwpb.Coin{},
		allocateds:      map[string]*allocatedmwpb.Coupon{},
		appusers:        map[string]*appusermwpb.User{},
		reviews:         map[string]*reviewmwpb.Review{},
	}

	if err := handler.getAppCoins(ctx); err != nil {
		return nil, err
	}
	if err := handler.getReviews(ctx); err != nil {
		return nil, err
	}
	if err := handler.getAllocateds(ctx); err != nil {
		return nil, err
	}
	if err := handler.getAppUsers(ctx); err != nil {
		return nil, err
	}

	handler.formalize()
	if len(handler.infos) == 0 {
		return nil, nil
	}
	return handler.infos[0], nil
}
