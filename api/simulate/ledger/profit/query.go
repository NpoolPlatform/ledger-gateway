package profit

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/simulate/ledger/profit"

	handler1 "github.com/NpoolPlatform/ledger-gateway/pkg/simulate/ledger/handler"
	profit1 "github.com/NpoolPlatform/ledger-gateway/pkg/simulate/ledger/profit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetProfits(ctx context.Context, in *npool.GetProfitsRequest) (*npool.GetProfitsResponse, error) {
	handler, err := profit1.NewHandler(
		ctx,
		handler1.WithAppID(&in.AppID, true),
		handler1.WithUserID(&in.UserID, true),
		handler1.WithOffset(in.Offset),
		handler1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetProfits",
			"In", in,
			"Error", err,
		)
		return &npool.GetProfitsResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetProfits(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetProfits",
			"In", in,
			"Error", err,
		)
		return &npool.GetProfitsResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetProfitsResponse{
		Infos: infos,
		Total: total,
	}, nil
}

func (s *Server) GetMiningRewards(ctx context.Context, in *npool.GetMiningRewardsRequest) (*npool.GetMiningRewardsResponse, error) {
	handler, err := profit1.NewHandler(
		ctx,
		handler1.WithAppID(&in.AppID, true),
		handler1.WithUserID(&in.UserID, true),
		handler1.WithStartAt(in.StartAt),
		handler1.WithEndAt(in.EndAt),
		handler1.WithOffset(in.Offset),
		handler1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetMiningRewards",
			"In", in,
			"Error", err,
		)
		return &npool.GetMiningRewardsResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetMiningRewards(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetMiningRewards",
			"In", in,
			"Error", err,
		)
		return &npool.GetMiningRewardsResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetMiningRewardsResponse{
		Infos: infos,
		Total: total,
	}, nil
}
