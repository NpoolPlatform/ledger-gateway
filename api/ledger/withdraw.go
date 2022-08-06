//nolint:nolintlint,dupl
package ledger

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledger1 "github.com/NpoolPlatform/ledger-gateway/pkg/ledger"

	"github.com/google/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetWithdraws(ctx context.Context, in *npool.GetWithdrawsRequest) (*npool.GetWithdrawsResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetWithdraws", "AppID", in.GetAppID(), "error", err)
		return &npool.GetWithdrawsResponse{}, status.Error(codes.InvalidArgument, "AppID is invalid")
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetWithdraws", "UserID", in.GetUserID(), "error", err)
		return &npool.GetWithdrawsResponse{}, status.Error(codes.InvalidArgument, "UserID is invalid")
	}

	infos, n, err := ledger1.GetWithdraws(ctx, in.GetAppID(), in.GetUserID(), in.GetOffset(), in.GetLimit())
	if err != nil {
		logger.Sugar().Errorw("GetWithdraws", "error", err)
		return &npool.GetWithdrawsResponse{}, status.Error(codes.Internal, "fail get withdraws")
	}

	return &npool.GetWithdrawsResponse{
		Infos: infos,
		Total: n,
	}, nil
}

func (s *Server) GetIntervalWithdraws(
	ctx context.Context, in *npool.GetIntervalWithdrawsRequest,
) (
	*npool.GetIntervalWithdrawsResponse, error,
) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetIntervalWithdraws", "AppID", in.GetAppID(), "error", err)
		return &npool.GetIntervalWithdrawsResponse{}, status.Error(codes.InvalidArgument, "AppID is invalid")
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetIntervalWithdraws", "UserID", in.GetUserID(), "error", err)
		return &npool.GetIntervalWithdrawsResponse{}, status.Error(codes.InvalidArgument, "UserID is invalid")
	}

	infos, n, err := ledger1.GetIntervalWithdraws(
		ctx,
		in.GetAppID(), in.GetUserID(),
		in.GetStartAt(), in.GetEndAt(),
		in.GetOffset(), in.GetLimit(),
	)
	if err != nil {
		logger.Sugar().Errorw("GetIntervalWithdraws", "error", err)
		return &npool.GetIntervalWithdrawsResponse{}, status.Error(codes.Internal, "fail get withdraws")
	}

	return &npool.GetIntervalWithdrawsResponse{
		Infos: infos,
		Total: n,
	}, nil
}
