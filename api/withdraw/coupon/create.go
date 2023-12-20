package coupon

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw/coupon"

	couponwithdraw1 "github.com/NpoolPlatform/ledger-gateway/pkg/withdraw/coupon"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateCouponWithdraw(ctx context.Context, in *npool.CreateCouponWithdrawRequest) (*npool.CreateCouponWithdrawResponse, error) {
	handler, err := couponwithdraw1.NewHandler(
		ctx,
		couponwithdraw1.WithAppID(&in.AppID, true),
		couponwithdraw1.WithUserID(&in.UserID, true),
		couponwithdraw1.WithAllocatedID(&in.AllocatedID, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateCouponWithdraw",
			"In", in,
			"Error", err,
		)
		return &npool.CreateCouponWithdrawResponse{}, status.Error(codes.Aborted, err.Error())
	}

	info, err := handler.CreateCouponWithdraw(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateCouponWithdraw",
			"In", in,
			"Error", err,
		)
		return &npool.CreateCouponWithdrawResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.CreateCouponWithdrawResponse{
		Info: info,
	}, nil
}
