package transfer

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	transfer1 "github.com/NpoolPlatform/ledger-gateway/pkg/transfer"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/transfer"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateTransfer(ctx context.Context, in *npool.CreateTransferRequest) (
	resp *npool.CreateTransferResponse,
	err error,
) {
	handler, err := transfer1.NewHandler(
		ctx,
		transfer1.WithAppID(&in.AppID, true),
		transfer1.WithUserID(&in.AppID, &in.UserID, true),
		transfer1.WithAccount(&in.Account, true),
		transfer1.WithAccountType(&in.AccountType, true),
		transfer1.WithVerificationCode(&in.VerificationCode, true),
		transfer1.WithCoinTypeID(&in.AppID, &in.CoinTypeID, true),
		transfer1.WithTargetUserID(&in.AppID, &in.TargetUserID, true),
		transfer1.WithAmount(&in.Amount, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateTransfer",
			"In", in,
			"Error", err,
		)
		return &npool.CreateTransferResponse{}, status.Error(codes.Aborted, err.Error())
	}

	info, err := handler.CreateTransfer(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateTransfer",
			"In", in,
			"Error", err,
		)
		return &npool.CreateTransferResponse{}, status.Error(codes.Aborted, err.Error())
	}

	return &npool.CreateTransferResponse{
		Info: info,
	}, nil
}
