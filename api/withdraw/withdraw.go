//nolint:nolintlint,dupl
package ledger

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	ledger1 "github.com/NpoolPlatform/ledger-gateway/pkg/ledger"

	"github.com/google/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateWithdraw(ctx context.Context, in *npool.CreateWithdrawRequest) (*npool.CreateWithdrawResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("validate", "AppID", in.GetUserID(), "error", err)
		return &npool.CreateWithdrawResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("AppID is invalid: %v", err))
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("validate", "UserID", in.GetUserID(), "error", err)
		return &npool.CreateWithdrawResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("UserID is invalid: %v", err))
	}

	if _, err := uuid.Parse(in.GetCoinTypeID()); err != nil {
		logger.Sugar().Errorw("validate", "CoinTypeID", in.GetCoinTypeID(), "error", err)
		return &npool.CreateWithdrawResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("CoinTypeID is invalid: %v", err))
	}

	if _, err := uuid.Parse(in.GetAccountID()); err != nil {
		logger.Sugar().Errorw("validate", "AccountID", in.GetAccountID(), "error", err)
		return &npool.CreateWithdrawResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("AccountID is invalid: %v", err))
	}

	if _, err := decimal.NewFromString(in.GetAmount()); err != nil {
		logger.Sugar().Errorw("validate", "Amount", in.GetAmount(), "error", err)
		return &npool.CreateWithdrawResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("Amount is invalid: %v", err))
	}

	amount := decimal.RequireFromString(in.GetAmount())
	if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
		logger.Sugar().Errorw("validate", "Amount", in.GetAmount())
		return &npool.CreateWithdrawResponse{}, status.Error(codes.InvalidArgument, "Amount is less than 0")
	}

	info, err := ledger1.CreateWithdraw(
		ctx,
		in.GetAppID(), in.GetUserID(),
		in.GetCoinTypeID(), in.GetAccountID(),
		amount,
		in.GetAccountType(),
		in.GetAccount(),
		in.GetVerificationCode(),
	)
	if err != nil {
		logger.Sugar().Errorw("CreateWithdraw", "error", err)
		return &npool.CreateWithdrawResponse{}, status.Error(codes.Internal, "fail create withdraw")
	}

	return &npool.CreateWithdrawResponse{
		Info: info,
	}, nil
}

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

func (s *Server) GetAppWithdraws(ctx context.Context, in *npool.GetAppWithdrawsRequest) (*npool.GetAppWithdrawsResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetAppWithdraws", "AppID", in.GetAppID(), "error", err)
		return &npool.GetAppWithdrawsResponse{}, status.Error(codes.InvalidArgument, "AppID is invalid")
	}

	infos, n, err := ledger1.GetAppWithdraws(ctx, in.GetAppID(), in.GetOffset(), in.GetLimit())
	if err != nil {
		logger.Sugar().Errorw("GetAppWithdraws", "error", err)
		return &npool.GetAppWithdrawsResponse{}, status.Error(codes.Internal, "fail get withdraws")
	}

	return &npool.GetAppWithdrawsResponse{
		Infos: infos,
		Total: n,
	}, nil
}

func (s *Server) GetNAppWithdraws(ctx context.Context, in *npool.GetNAppWithdrawsRequest) (*npool.GetNAppWithdrawsResponse, error) {
	resp, err := s.GetAppWithdraws(ctx, &npool.GetAppWithdrawsRequest{
		AppID:  in.TargetAppID,
		Offset: in.Offset,
		Limit:  in.Limit,
	})
	if err != nil {
		return &npool.GetNAppWithdrawsResponse{}, err
	}

	return &npool.GetNAppWithdrawsResponse{
		Infos: resp.Infos,
		Total: resp.Total,
	}, nil
}
