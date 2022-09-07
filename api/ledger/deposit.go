package ledger

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	mledger "github.com/NpoolPlatform/ledger-gateway/pkg/ledger"
	constant "github.com/NpoolPlatform/ledger-gateway/pkg/message/const"
	"github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel"
	scodes "go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateAppUserDeposit(
	ctx context.Context,
	in *ledger.CreateAppUserDepositRequest,
) (
	resp *ledger.CreateAppUserDepositResponse,
	err error,
) {
	_, span := otel.Tracer(constant.ServiceName).Start(ctx, "CreateAppUserDeposit")
	defer span.End()

	defer func() {
		if err != nil {
			span.SetStatus(scodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	if _, err := uuid.Parse(in.GetTargetUserID()); err != nil {
		logger.Sugar().Errorw("CreateAppUserDeposit", "TargetUserID", in.GetTargetUserID(), "error", err)
		return &ledger.CreateAppUserDepositResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetTargetUserID()); err != nil {
		logger.Sugar().Errorw("CreateAppUserDeposit", "TargetUserID", in.GetTargetUserID(), "error", err)
		return &ledger.CreateAppUserDepositResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetCoinTypeID()); err != nil {
		logger.Sugar().Errorw("CreateAppUserDeposit", "CoinTypeID", in.GetCoinTypeID(), "error", err)
		return &ledger.CreateAppUserDepositResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("CoinTypeID is invalid: %v", err))
	}

	if _, err := decimal.NewFromString(in.GetAmount()); err != nil {
		logger.Sugar().Errorw("CreateAppUserDeposit", "Amount", in.GetAmount(), "error", err)
		return &ledger.CreateAppUserDepositResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("Amount is invalid: %v", err))
	}

	amount := decimal.RequireFromString(in.GetAmount())
	if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
		logger.Sugar().Errorw("CreateAppUserDeposit", "Amount", in.GetAmount())
		return &ledger.CreateAppUserDepositResponse{}, status.Error(codes.InvalidArgument, "Amount is less than 0")
	}

	if _, err := uuid.Parse(in.GetDepositAppID()); err != nil {
		logger.Sugar().Errorw("CreateAppUserDeposit", "DepositAppID", in.GetDepositAppID(), "error", err)
		return &ledger.CreateAppUserDepositResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetDepositUserID()); err != nil {
		logger.Sugar().Errorw("CreateAppUserDeposit", "DepositUserID", in.GetDepositUserID(), "error", err)
		return &ledger.CreateAppUserDepositResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := mledger.CreateDeposit(
		ctx,
		in.GetTargetAppID(),
		in.GetTargetUserID(),
		in.GetCoinTypeID(),
		in.GetAmount(),
		in.GetDepositAppID(),
		in.GetDepositUserID())
	if err != nil {
		logger.Sugar().Errorw("CreateAppUserDeposit", "error", err)
		return &ledger.CreateAppUserDepositResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &ledger.CreateAppUserDepositResponse{
		Info: info,
	}, nil
}
