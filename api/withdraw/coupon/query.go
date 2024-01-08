//nolint:nolintlint,dupl
package coupon

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	couponwithdraw1 "github.com/NpoolPlatform/ledger-gateway/pkg/withdraw/coupon"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw/coupon"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetCouponWithdraws(ctx context.Context, in *npool.GetCouponWithdrawsRequest) (*npool.GetCouponWithdrawsResponse, error) {
	handler, err := couponwithdraw1.NewHandler(
		ctx,
		couponwithdraw1.WithAppID(&in.AppID, true),
		couponwithdraw1.WithUserID(&in.UserID, true),
		couponwithdraw1.WithOffset(in.Offset),
		couponwithdraw1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetCouponWithdraws",
			"In", in,
			"Error", err,
		)
		return &npool.GetCouponWithdrawsResponse{}, status.Error(codes.Aborted, err.Error())
	}

	infos, total, err := handler.GetCouponWithdraws(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetCouponWithdraws",
			"In", in,
			"Error", err,
		)
		return &npool.GetCouponWithdrawsResponse{}, status.Error(codes.Aborted, err.Error())
	}

	return &npool.GetCouponWithdrawsResponse{
		Infos: infos,
		Total: total,
	}, nil
}

func (s *Server) GetAppCouponWithdraws(ctx context.Context, in *npool.GetAppCouponWithdrawsRequest) (*npool.GetAppCouponWithdrawsResponse, error) {
	handler, err := couponwithdraw1.NewHandler(
		ctx,
		couponwithdraw1.WithAppID(&in.AppID, true),
		couponwithdraw1.WithOffset(in.Offset),
		couponwithdraw1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetAppCouponWithdraws",
			"In", in,
			"Error", err,
		)
		return &npool.GetAppCouponWithdrawsResponse{}, status.Error(codes.Aborted, err.Error())
	}

	infos, total, err := handler.GetCouponWithdraws(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetAppCouponWithdraws",
			"In", in,
			"Error", err,
		)
		return &npool.GetAppCouponWithdrawsResponse{}, status.Error(codes.Aborted, err.Error())
	}

	return &npool.GetAppCouponWithdrawsResponse{
		Infos: infos,
		Total: total,
	}, nil
}
