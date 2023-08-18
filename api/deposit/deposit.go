package deposit

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	deposit1 "github.com/NpoolPlatform/ledger-gateway/pkg/deposit"
	"github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateAppUserDeposit(ctx context.Context, in *npool.CreateAppUserDepositRequest) (
	resp *npool.CreateAppUserDepositResponse,
	err error,
) {
	handler, err := deposit1.NewHandler(
		ctx,
		deposit1.WithAppID(&in.AppID, true),
		deposit1.WithUserID(&in.AppID, &in.UserID, true),
		deposit1.WithCoinTypeID(&in.TargetAppID, &in.CoinTypeID, true),
		deposit1.WithTargetAppID(&in.TargetAppID, true),
		deposit1.WithTargetUserID(&in.TargetAppID, &in.TargetUserID, true),
		deposit1.WithAmount(&in.Amount, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateAppUserDeposit",
			"In", in,
			"Error", err,
		)
		return &npool.CreateAppUserDepositResponse{}, status.Error(codes.Aborted, err.Error())
	}

	info, err := handler.CreateDeposit(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateAppUserDeposit",
			"In", in,
			"Error", err,
		)
		return &ledger.CreateAppUserDepositResponse{}, status.Error(codes.Aborted, err.Error())
	}

	return &ledger.CreateAppUserDepositResponse{
		Info: info,
	}, nil
}
