package withdraw

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw"

	withdraw1 "github.com/NpoolPlatform/ledger-gateway/pkg/withdraw"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateWithdraw(ctx context.Context, in *npool.CreateWithdrawRequest) (*npool.CreateWithdrawResponse, error) {
	handler, err := withdraw1.NewHandler(
		ctx,
		withdraw1.WithAppID(&in.AppID, true),
		withdraw1.WithUserID(&in.AppID, &in.UserID, true),
		withdraw1.WithAccount(&in.Account, true),
		withdraw1.WithAccountType(&in.AccountType, true),
		withdraw1.WithVerificationCode(&in.VerificationCode, true),
		withdraw1.WithCoinTypeID(&in.AppID, &in.CoinTypeID, true),
		withdraw1.WithAccountID(&in.AppID, &in.AccountID, true),
		withdraw1.WithAmount(&in.Amount, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateWithdraw",
			"In", in,
			"Error", err,
		)
		return &npool.CreateWithdrawResponse{}, status.Error(codes.Aborted, err.Error())
	}

	info, err := handler.CreateWithdraw(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateWithdraw",
			"In", in,
			"Error", err,
		)
		return &npool.CreateWithdrawResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.CreateWithdrawResponse{
		Info: info,
	}, nil
}
