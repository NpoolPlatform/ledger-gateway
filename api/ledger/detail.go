//nolint:nolintlint,dupl
package ledger

import (
	"context"
	"time"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledger1 "github.com/NpoolPlatform/ledger-gateway/pkg/ledger"

	"github.com/google/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetDetails(ctx context.Context, in *npool.GetDetailsRequest) (*npool.GetDetailsResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetDetails", "AppID", in.GetAppID(), "error", err)
		return &npool.GetDetailsResponse{}, status.Error(codes.InvalidArgument, "AppID is invalid")
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetDetails", "UserID", in.GetUserID(), "error", err)
		return &npool.GetDetailsResponse{}, status.Error(codes.InvalidArgument, "UserID is invalid")
	}

	endAt := in.GetEndAt()
	if endAt == 0 {
		endAt = uint32(time.Now().Unix())
	}

	infos, n, err := ledger1.GetDetails(
		ctx,
		in.GetAppID(), in.GetUserID(),
		in.GetStartAt(), endAt,
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

func (s *Server) GetMiningRewards(ctx context.Context, in *npool.GetMiningRewardsRequest) (*npool.GetMiningRewardsResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetMiningRewards", "AppID", in.GetAppID(), "error", err)
		return &npool.GetMiningRewardsResponse{}, status.Error(codes.InvalidArgument, "AppID is invalid")
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetMiningRewards", "UserID", in.GetUserID(), "error", err)
		return &npool.GetMiningRewardsResponse{}, status.Error(codes.InvalidArgument, "UserID is invalid")
	}

	endAt := in.GetEndAt()
	if endAt == 0 {
		endAt = uint32(time.Now().Unix())
	}

	infos, n, err := ledger1.GetMiningRewards(
		ctx,
		in.GetAppID(), in.GetUserID(),
		in.GetStartAt(), endAt,
		in.GetOffset(), in.GetLimit(),
	)
	if err != nil {
		logger.Sugar().Errorw("GetMiningRewards", "error", err)
		return &npool.GetMiningRewardsResponse{}, status.Error(codes.Internal, "fail get mining rewards")
	}

	return &npool.GetMiningRewardsResponse{
		Infos: infos,
		Total: n,
	}, nil
}

func (s *Server) GetAppDetails(ctx context.Context, in *npool.GetAppDetailsRequest) (*npool.GetAppDetailsResponse, error) {
	if _, err := uuid.Parse(in.GetTargetAppID()); err != nil {
		logger.Sugar().Errorw("GetAppDetails", "TargetAppID", in.GetTargetAppID(), "error", err)
		return &npool.GetAppDetailsResponse{}, status.Error(codes.InvalidArgument, "TargetAppID is invalid")
	}

	infos, n, err := ledger1.GetAppDetails(
		ctx,
		in.GetTargetAppID(),
		in.GetOffset(),
		in.GetLimit(),
	)
	if err != nil {
		logger.Sugar().Errorw("GetAppDetails", "error", err)
		return &npool.GetAppDetailsResponse{}, status.Error(codes.Internal, "fail get app details")
	}

	return &npool.GetAppDetailsResponse{
		Infos: infos,
		Total: n,
	}, nil
}
