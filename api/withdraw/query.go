//nolint:nolintlint,dupl
package withdraw

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	withdraw1 "github.com/NpoolPlatform/ledger-gateway/pkg/withdraw"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetWithdraws(ctx context.Context, in *npool.GetWithdrawsRequest) (*npool.GetWithdrawsResponse, error) {
	handler, err := withdraw1.NewHandler(
		ctx,
		withdraw1.WithAppID(&in.AppID, true),
		withdraw1.WithUserID(&in.AppID, &in.UserID, true),
		withdraw1.WithOffset(in.Offset),
		withdraw1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetWithdraws",
			"In", in,
			"Error", err,
		)
		return &npool.GetWithdrawsResponse{}, status.Error(codes.Aborted, err.Error())
	}

	infos, total, err := handler.GetWithdraws(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetWithdraws",
			"In", in,
			"Error", err,
		)
		return &npool.GetWithdrawsResponse{}, status.Error(codes.Aborted, err.Error())
	}

	return &npool.GetWithdrawsResponse{
		Infos: infos,
		Total: total,
	}, nil
}

func (s *Server) GetAppWithdraws(ctx context.Context, in *npool.GetAppWithdrawsRequest) (*npool.GetAppWithdrawsResponse, error) {
	handler, err := withdraw1.NewHandler(
		ctx,
		withdraw1.WithAppID(&in.AppID, true),
		withdraw1.WithOffset(in.Offset),
		withdraw1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetAppWithdraws",
			"In", in,
			"Error", err,
		)
		return &npool.GetAppWithdrawsResponse{}, status.Error(codes.Aborted, err.Error())
	}

	infos, total, err := handler.GetWithdraws(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetAppWithdraws",
			"In", in,
			"Error", err,
		)
		return &npool.GetAppWithdrawsResponse{}, status.Error(codes.Aborted, err.Error())
	}

	return &npool.GetAppWithdrawsResponse{
		Infos: infos,
		Total: total,
	}, nil
}

func (s *Server) GetNAppWithdraws(ctx context.Context, in *npool.GetNAppWithdrawsRequest) (*npool.GetNAppWithdrawsResponse, error) {
	resp, err := s.GetAppWithdraws(ctx, &npool.GetAppWithdrawsRequest{
		AppID:  in.TargetAppID,
		Offset: in.Offset,
		Limit:  in.Limit,
	})
	if err != nil {
		logger.Sugar().Errorw(
			"GetAppWithdraws",
			"In", in,
			"Error", err,
		)
		return &npool.GetNAppWithdrawsResponse{}, status.Error(codes.Aborted, err.Error())
	}

	return &npool.GetNAppWithdrawsResponse{
		Infos: resp.Infos,
		Total: resp.Total,
	}, nil
}
