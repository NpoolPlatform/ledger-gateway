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

func (s *Server) GetProfits(ctx context.Context, in *npool.GetProfitsRequest) (*npool.GetProfitsResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetProfits", "AppID", in.GetAppID(), "error", err)
		return &npool.GetProfitsResponse{}, status.Error(codes.InvalidArgument, "AppID is invalid")
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetProfits", "UserID", in.GetUserID(), "error", err)
		return &npool.GetProfitsResponse{}, status.Error(codes.InvalidArgument, "UserID is invalid")
	}

	infos, n, err := ledger1.GetProfits(ctx, in.GetAppID(), in.GetUserID(), in.GetOffset(), in.GetLimit())
	if err != nil {
		logger.Sugar().Errorw("GetProfits", "error", err)
		return &npool.GetProfitsResponse{}, status.Error(codes.Internal, "fail get generals")
	}

	return &npool.GetProfitsResponse{
		Infos: infos,
		Total: n,
	}, nil
}

func (s *Server) GetIntervalProfits(
	ctx context.Context, in *npool.GetIntervalProfitsRequest,
) (
	*npool.GetIntervalProfitsResponse, error,
) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetIntervalProfits", "AppID", in.GetAppID(), "error", err)
		return &npool.GetIntervalProfitsResponse{}, status.Error(codes.InvalidArgument, "AppID is invalid")
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetIntervalProfits", "UserID", in.GetUserID(), "error", err)
		return &npool.GetIntervalProfitsResponse{}, status.Error(codes.InvalidArgument, "UserID is invalid")
	}

	infos, n, err := ledger1.GetIntervalProfits(
		ctx,
		in.GetAppID(), in.GetUserID(),
		in.GetStartAt(), in.GetEndAt(),
		in.GetOffset(), in.GetLimit(),
	)
	if err != nil {
		logger.Sugar().Errorw("GetIntervalProfits", "error", err)
		return &npool.GetIntervalProfitsResponse{}, status.Error(codes.Internal, "fail get generals")
	}

	return &npool.GetIntervalProfitsResponse{
		Infos: infos,
		Total: n,
	}, nil
}

func (s *Server) GetGoodProfits(
	ctx context.Context, in *npool.GetGoodProfitsRequest,
) (
	*npool.GetGoodProfitsResponse, error,
) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetGoodProfits", "AppID", in.GetAppID(), "error", err)
		return &npool.GetGoodProfitsResponse{}, status.Error(codes.InvalidArgument, "AppID is invalid")
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetGoodProfits", "UserID", in.GetUserID(), "error", err)
		return &npool.GetGoodProfitsResponse{}, status.Error(codes.InvalidArgument, "UserID is invalid")
	}

	if in.GetLimit() <= 0 {
		logger.Sugar().Errorw("limit is less than or equal to 0")
		return &npool.GetGoodProfitsResponse{}, status.Error(codes.InvalidArgument, "limit is less than or equal to 0")
	}
	infos, n, err := ledger1.GetGoodProfits(
		ctx,
		in.GetAppID(), in.GetUserID(),
		in.GetStartAt(), in.GetEndAt(),
		in.GetOffset(), in.GetLimit(),
	)
	if err != nil {
		logger.Sugar().Errorw("GetGoodProfits", "error", err)
		return &npool.GetGoodProfitsResponse{}, status.Error(codes.Internal, "fail get generals")
	}

	return &npool.GetGoodProfitsResponse{
		Infos: infos,
		Total: n,
	}, nil
}
