package statement

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	handler1 "github.com/NpoolPlatform/ledger-gateway/pkg/ledger/handler"
	statement1 "github.com/NpoolPlatform/ledger-gateway/pkg/ledger/statement"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetStatements(ctx context.Context, in *npool.GetStatementsRequest) (*npool.GetStatementsResponse, error) {
	handler, err := statement1.NewHandler(
		ctx,
		handler1.WithAppID(&in.AppID, true),
		handler1.WithUserID(&in.AppID, &in.UserID, true),
		handler1.WithOffset(in.Offset),
		handler1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetStatements",
			"In", in,
			"Error", err,
		)
		return &npool.GetStatementsResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetStatements(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetStatements",
			"In", in,
			"Error", err,
		)
		return &npool.GetStatementsResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetStatementsResponse{
		Infos: infos,
		Total: total,
	}, nil
}

func (s *Server) GetAppStatements(ctx context.Context, in *npool.GetAppStatementsRequest) (*npool.GetAppStatementsResponse, error) {
	handler, err := statement1.NewHandler(
		ctx,
		handler1.WithAppID(&in.TargetAppID, true),
		handler1.WithOffset(in.Offset),
		handler1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetAppStatements",
			"In", in,
			"Error", err,
		)
		return &npool.GetAppStatementsResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetStatements(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetAppStatements",
			"In", in,
			"Error", err,
		)
		return &npool.GetAppStatementsResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetAppStatementsResponse{
		Infos: infos,
		Total: total,
	}, nil
}
