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

func (s *Server) GetGenerals(ctx context.Context, in *npool.GetGeneralsRequest) (*npool.GetGeneralsResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetGenerals", "AppID", in.GetAppID(), "error", err)
		return &npool.GetGeneralsResponse{}, status.Error(codes.InvalidArgument, "AppID is invalid")
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetGenerals", "UserID", in.GetUserID(), "error", err)
		return &npool.GetGeneralsResponse{}, status.Error(codes.InvalidArgument, "UserID is invalid")
	}

	infos, n, err := ledger1.GetGenerals(ctx, in.GetAppID(), in.GetUserID(), in.GetOffset(), in.GetLimit())
	if err != nil {
		logger.Sugar().Errorw("GetGenerals", "error", err)
		return &npool.GetGeneralsResponse{}, status.Error(codes.Internal, "fail get generals")
	}

	return &npool.GetGeneralsResponse{
		Infos: infos,
		Total: n,
	}, nil
}

func (s *Server) GetIntervalGenerals(
	ctx context.Context, in *npool.GetIntervalGeneralsRequest,
) (
	*npool.GetIntervalGeneralsResponse, error,
) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetIntervalGenerals", "AppID", in.GetAppID(), "error", err)
		return &npool.GetIntervalGeneralsResponse{}, status.Error(codes.InvalidArgument, "AppID is invalid")
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetIntervalGenerals", "UserID", in.GetUserID(), "error", err)
		return &npool.GetIntervalGeneralsResponse{}, status.Error(codes.InvalidArgument, "UserID is invalid")
	}

	infos, n, err := ledger1.GetIntervalGenerals(
		ctx,
		in.GetAppID(), in.GetUserID(),
		in.GetStartAt(), in.GetEndAt(),
		in.GetOffset(), in.GetLimit(),
	)
	if err != nil {
		logger.Sugar().Errorw("GetIntervalGenerals", "error", err)
		return &npool.GetIntervalGeneralsResponse{}, status.Error(codes.Internal, "fail get generals")
	}

	return &npool.GetIntervalGeneralsResponse{
		Infos: infos,
		Total: n,
	}, nil
}

func (s *Server) GetDetails(ctx context.Context, in *npool.GetDetailsRequest) (*npool.GetDetailsResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetDetails", "AppID", in.GetAppID(), "error", err)
		return &npool.GetDetailsResponse{}, status.Error(codes.InvalidArgument, "AppID is invalid")
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetDetails", "UserID", in.GetUserID(), "error", err)
		return &npool.GetDetailsResponse{}, status.Error(codes.InvalidArgument, "UserID is invalid")
	}

	infos, n, err := ledger1.GetDetails(
		ctx,
		in.GetAppID(), in.GetUserID(),
		in.GetStartAt(), in.GetEndAt(),
		in.GetOffset(), in.GetLimit(),
	)
	if err != nil {
		logger.Sugar().Errorw("GetDetails", "error", err)
		return &npool.GetDetailsResponse{}, status.Error(codes.Internal, "fail get generals")
	}

	return &npool.GetDetailsResponse{
		Infos: infos,
		Total: n,
	}, nil
}

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
