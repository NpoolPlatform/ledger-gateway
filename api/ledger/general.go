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

func (s *Server) GetAppGenerals(ctx context.Context, in *npool.GetAppGeneralsRequest) (*npool.GetAppGeneralsResponse, error) {
	if _, err := uuid.Parse(in.GetTargetAppID()); err != nil {
		logger.Sugar().Errorw("GetAppGenerals", "TargetAppID", in.GetTargetAppID(), "error", err)
		return &npool.GetAppGeneralsResponse{}, status.Error(codes.InvalidArgument, "TargetAppID is invalid")
	}

	infos, n, err := ledger1.GetAppGenerals(ctx, in.GetTargetAppID(), in.GetOffset(), in.GetLimit())
	if err != nil {
		logger.Sugar().Errorw("GetAppGenerals", "error", err)
		return &npool.GetAppGeneralsResponse{}, status.Error(codes.Internal, "fail get app generals")
	}

	return &npool.GetAppGeneralsResponse{
		Infos: infos,
		Total: n,
	}, nil
}
