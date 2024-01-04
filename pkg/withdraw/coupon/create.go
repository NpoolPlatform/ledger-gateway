package coupon

import (
	"context"
	"fmt"
	"time"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	couponmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon"
	allocatedmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	cashcontrolmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/app/cashcontrol"
	couponcoinmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/app/coin"
	constant "github.com/NpoolPlatform/ledger-gateway/pkg/const"
	ledgergwname "github.com/NpoolPlatform/ledger-gateway/pkg/servicename"
	ledgermwname "github.com/NpoolPlatform/ledger-middleware/pkg/servicename"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	inspiretypes "github.com/NpoolPlatform/message/npool/basetypes/inspire/v1"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	reviewtypes "github.com/NpoolPlatform/message/npool/basetypes/review/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	cashcontrolmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/app/cashcontrol"
	couponcoinmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/app/coin"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw/coupon"
	couponwithdrawmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/withdraw/coupon"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	reviewmwpb "github.com/NpoolPlatform/message/npool/review/mw/v2/review"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	reviewsvcname "github.com/NpoolPlatform/review-middleware/pkg/servicename"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type createHandler struct {
	*Handler
	ReviewID *string
	CouponID *string
	user     *usermwpb.User
}

func (h *createHandler) getUser(ctx context.Context) error {
	user, err := usermwcli.GetUser(ctx, *h.AppID, *h.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("invalid user")
	}
	h.user = user
	return nil
}

func (h *createHandler) checkKyc() error {
	if h.user.State != basetypes.KycState_Approved {
		return fmt.Errorf("kyc not approved")
	}
	return nil
}

func (h *createHandler) checkCreditThreshold(value string) error {
	credits, err := decimal.NewFromString(h.user.ActionCredits)
	if err != nil {
		return err
	}
	if credits.Cmp(decimal.RequireFromString(value)) < 0 {
		return fmt.Errorf("credits not enough")
	}
	return nil
}

func (h *createHandler) checkPaymentAmountThreshold(ctx context.Context, value string) error {
	amounts, err := ordermwcli.SumOrderPaymentAmounts(ctx, &ordermwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		OrderStates: &basetypes.Uint32SliceVal{Op: cruder.IN, Value: []uint32{
			uint32(ordertypes.OrderState_OrderStatePaid),
			uint32(ordertypes.OrderState_OrderStateInService),
			uint32(ordertypes.OrderState_OrderStateExpired),
		}},
	})
	if err != nil {
		return err
	}
	if decimal.RequireFromString(amounts).Cmp(decimal.RequireFromString(value)) < 0 {
		return fmt.Errorf("payment amounts not enough")
	}
	return nil
}

func (h *createHandler) checkOrderThreshold(ctx context.Context, value string) error {
	total, err := ordermwcli.CountOrders(ctx, &ordermwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		OrderStates: &basetypes.Uint32SliceVal{Op: cruder.IN, Value: []uint32{
			uint32(ordertypes.OrderState_OrderStatePaid),
			uint32(ordertypes.OrderState_OrderStateInService),
			uint32(ordertypes.OrderState_OrderStateExpired),
		}},
	})
	if err != nil {
		return err
	}

	_total := decimal.NewFromInt32(int32(total))
	_value := decimal.RequireFromString(value)
	if _value.Cmp(decimal.RequireFromString("0")) == 0 { // first order
		if !_total.Equal(decimal.RequireFromString("0")) {
			return fmt.Errorf("you have already purchased")
		}
	}
	if _total.Cmp(_value) < 0 {
		return fmt.Errorf("not enough orders")
	}
	return nil
}

func (h *createHandler) checkAllocated(ctx context.Context) error {
	allocated, err := allocatedmwcli.GetCouponOnly(ctx, &allocatedmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		EntID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AllocatedID},
		Used:   &basetypes.BoolVal{Op: cruder.EQ, Value: false},
	})
	if err != nil {
		return err
	}
	if allocated == nil {
		return fmt.Errorf("invalid coupon")
	}
	if !allocated.Cashable {
		return fmt.Errorf("permission denied")
	}
	if allocated.CouponType != inspiretypes.CouponType_FixAmount {
		return fmt.Errorf("invaild coupon type")
	}

	now := uint32(time.Now().Unix())
	if now < allocated.StartAt || now > allocated.EndAt {
		return fmt.Errorf("coupon can not be withdraw in current time")
	}

	h.Amount = &allocated.Denomination
	h.CouponID = &allocated.CouponID
	return nil
}

func (h *createHandler) checkCoupon(ctx context.Context) error {
	coupon, err := couponmwcli.GetCoupon(ctx, *h.CouponID)
	if err != nil {
		return err
	}
	if coupon == nil {
		return fmt.Errorf("invalid coupon")
	}
	if coupon.CouponType != inspiretypes.CouponType_FixAmount {
		return fmt.Errorf("invaild coupon type")
	}
	return nil
}

func (h *createHandler) getCouponCoin(ctx context.Context) error {
	info, err := couponcoinmwcli.GetCouponCoinOnly(ctx, &couponcoinmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	})
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("couponcoin not found")
	}
	h.CoinTypeID = &info.CoinTypeID
	return nil
}

func (h *createHandler) checkoutCouponControl(ctx context.Context) error {
	coupon, err := couponmwcli.GetCoupon(ctx, *h.CouponID)
	if err != nil {
		return err
	}
	if coupon == nil {
		return fmt.Errorf("invalid coupon")
	}

	offset := int32(0)
	limit := constant.DefaultRowLimit
	for {
		controls, _, err := cashcontrolmwcli.GetCashControls(ctx, &cashcontrolmwpb.Conds{
			AppID:    &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			CouponID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.CouponID},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(controls) == 0 {
			return nil
		}

		for _, control := range controls {
			if _, err := decimal.NewFromString(control.Value); err != nil {
				return err
			}
			var err error
			switch control.ControlType {
			case inspiretypes.ControlType_KycApproved:
				err = h.checkKyc()
			case inspiretypes.ControlType_CreditThreshold:
				err = h.checkCreditThreshold(control.Value)
			case inspiretypes.ControlType_OrderThreshold:
				err = h.checkOrderThreshold(ctx, control.Value)
			case inspiretypes.ControlType_PaymentAmountThreshold:
				err = h.checkPaymentAmountThreshold(ctx, control.Value)
			default:
				return fmt.Errorf("invalid control type")
			}
			if err != nil {
				return err
			}
		}
		offset += limit
	}
}

func (h *createHandler) withCreateCouponWithdraw(dispose *dtmcli.SagaDispose) {
	req := &couponwithdrawmwpb.CouponWithdrawReq{
		EntID:       h.EntID,
		AppID:       h.AppID,
		UserID:      h.UserID,
		CoinTypeID:  h.CoinTypeID,
		AllocatedID: h.AllocatedID,
		Amount:      h.Amount,
		ReviewID:    h.ReviewID,
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
	objectType := reviewtypes.ReviewObjectType_ObjectRandomCouponCash
	serviceDomain := ledgergwname.ServiceDomain
	req := &reviewmwpb.ReviewReq{
		EntID:      h.ReviewID,
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
		Handler: h,
	}
	if err := handler.getUser(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkAllocated(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkCoupon(ctx); err != nil {
		return nil, err
	}
	if err := handler.getCouponCoin(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkoutCouponControl(ctx); err != nil {
		return nil, err
	}

	id := uuid.NewString()
	if h.EntID == nil {
		h.EntID = &id
	}
	id1 := uuid.NewString()
	if handler.ReviewID == nil {
		handler.ReviewID = &id1
	}

	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		TimeoutToFail:  60,
		RequestTimeout: 10,
	})
	handler.withCreateCouponWithdraw(sagaDispose)
	handler.withCreateReview(sagaDispose)
	if err := dtmcli.WithSaga(ctx, sagaDispose); err != nil {
		return nil, err
	}
	return h.GetCouponWithdraw(ctx)
}
