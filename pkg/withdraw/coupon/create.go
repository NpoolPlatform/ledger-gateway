package coupon

import (
	"context"
	"fmt"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	allocatedmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	ledgergwname "github.com/NpoolPlatform/ledger-gateway/pkg/servicename"
	ledgermwname "github.com/NpoolPlatform/ledger-middleware/pkg/servicename"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	reviewtypes "github.com/NpoolPlatform/message/npool/basetypes/review/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw/coupon"
	couponwithdrawmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/withdraw/coupon"
	reviewmwpb "github.com/NpoolPlatform/message/npool/review/mw/v2/review"
	reviewsvcname "github.com/NpoolPlatform/review-middleware/pkg/servicename"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"

	"github.com/google/uuid"
)

type createHandler struct {
	*Handler
	user                  *usermwpb.User
	appCoin               *appcoinmwpb.Coin
	RequestTimeoutSeconds int64
}

func (h *createHandler) checkUser(ctx context.Context) error {
	user, err := usermwcli.GetUser(ctx, *h.AppID, *h.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("invalid user")
	}
	if user.State != basetypes.KycState_Approved {
		return fmt.Errorf("kyc not approved")
	}
	h.user = user
	return nil
}

func (h *createHandler) checkCoupon(ctx context.Context) error {
	allocated, err := allocatedmwcli.GetCouponOnly(ctx, &allocatedmwpb.Conds{
		AppID:    &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		CouponID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CouponID},
		Used:     &basetypes.BoolVal{Op: cruder.EQ, Value: false},
	})
	if err != nil {
		return err
	}
	if allocated == nil {
		return fmt.Errorf("invalid coupon")
	}
	h.Amount = &allocated.Denomination
	return nil
}

func (h *createHandler) getCoin(ctx context.Context) error {
	appCoin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CoinTypeID},
	})
	if err != nil {
		return err
	}
	if appCoin == nil {
		return fmt.Errorf("appcoin not found %v", *h.CoinTypeID)
	}
	h.appCoin = appCoin
	return nil
}

func (h *createHandler) withCreateCouponWithdraw(dispose *dtmcli.SagaDispose) {
	req := &couponwithdrawmwpb.CouponWithdrawReq{
		EntID:      h.EntID,
		AppID:      h.AppID,
		UserID:     h.UserID,
		CoinTypeID: h.CoinTypeID,
		CouponID:   h.CouponID,
		Amount:     h.Amount,
	}
	dispose.Add(
		ledgermwname.ServiceDomain,
		"ledger.middleware.withdraw.coupon.v2.Middleware/CreateCouponWithdraw",
		"ledger.middleware.withdraw.coupon.v2.Middleware/DeleteCouponWithdraw",
		&couponwithdrawmwpb.CreateCouponWithdrawRequest{
			Info: req,
		},
	)
}

func (h *createHandler) withCreateReview(dispose *dtmcli.SagaDispose) {
	id := uuid.NewString()
	objectType := reviewtypes.ReviewObjectType_ObjectCouponRandomCash
	serviceDomain := ledgergwname.ServiceDomain
	req := &reviewmwpb.ReviewReq{
		EntID:      &id,
		AppID:      h.AppID,
		ObjectID:   h.EntID,
		ObjectType: &objectType,
		Domain:     &serviceDomain,
	}
	dispose.Add(
		reviewsvcname.ServiceDomain,
		"review.middleware.review.v2.Middleware/CreateReview",
		"review.middleware.review.v2.Middleware/DeleteReview",
		&reviewmwpb.CreateReviewRequest{
			Info: req,
		},
	)
}

func (h *Handler) CreateCouponWithdraw(ctx context.Context) (*npool.CouponWithdraw, error) {
	handler := &createHandler{
		Handler:               h,
		RequestTimeoutSeconds: 10,
	}
	if err := handler.checkUser(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkCoupon(ctx); err != nil {
		return nil, err
	}
	if err := handler.getCoin(ctx); err != nil {
		return nil, err
	}

	id := uuid.NewString()
	if h.EntID == nil {
		h.EntID = &id
	}

	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		TimeoutToFail:  60,
		RequestTimeout: handler.RequestTimeoutSeconds,
	})
	handler.withCreateCouponWithdraw(sagaDispose)
	handler.withCreateReview(sagaDispose)

	if err := dtmcli.WithSaga(ctx, sagaDispose); err != nil {
		return nil, err
	}
	return h.GetCouponWithdraw(ctx)
}
