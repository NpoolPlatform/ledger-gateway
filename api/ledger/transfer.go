package ledger

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	constant "github.com/NpoolPlatform/ledger-gateway/pkg/message/const"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/otel"
	scodes "go.opentelemetry.io/otel/codes"

	"github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger"

	mledger "github.com/NpoolPlatform/ledger-gateway/pkg/ledger"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func (s *Server) CreateTransfer(ctx context.Context, in *ledger.CreateTransferRequest) (resp *ledger.CreateTransferResponse, err error) {
	_, span := otel.Tracer(constant.ServiceName).Start(ctx, "CreateTransfer")
	defer span.End()

	defer func() {
		if err != nil {
			span.SetStatus(scodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("CreateTransfer", "AppID", in.GetAppID(), "error", err)
		return &ledger.CreateTransferResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("CreateTransfer", "UserID", in.GetUserID(), "error", err)
		return &ledger.CreateTransferResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	switch in.GetAccountType() {
	case basetypes.SignMethod_Email, basetypes.SignMethod_Mobile:
		if in.GetAccount() == "" {
			logger.Sugar().Errorw("CreateTransfer", "Account empty", "Account", in.GetAccount())
			return &ledger.CreateTransferResponse{}, status.Error(codes.InvalidArgument, "Account id empty")
		}
	case basetypes.SignMethod_Google:
	default:
		logger.Sugar().Errorw("CreateTransfer", "AccountType empty", "AccountType", in.GetAccountType())
		return &ledger.CreateTransferResponse{}, status.Error(codes.InvalidArgument, "AccountType id invalid")
	}

	if in.GetVerificationCode() == "" {
		logger.Sugar().Errorw("CreateTransfer", "VerificationCode empty", "VerificationCode", in.GetVerificationCode())
		return &ledger.CreateTransferResponse{}, status.Error(codes.InvalidArgument, "VerificationCode id empty")
	}

	if _, err := uuid.Parse(in.GetTargetUserID()); err != nil {
		logger.Sugar().Errorw("CreateTransfer", "TransferUserID", in.GetTargetUserID(), "error", err)
		return &ledger.CreateTransferResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetCoinTypeID()); err != nil {
		logger.Sugar().Errorw("validate", "CoinTypeID", in.GetCoinTypeID(), "error", err)
		return &ledger.CreateTransferResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("CoinTypeID is invalid: %v", err))
	}

	if _, err := decimal.NewFromString(in.GetAmount()); err != nil {
		logger.Sugar().Errorw("validate", "Amount", in.GetAmount(), "error", err)
		return &ledger.CreateTransferResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("Amount is invalid: %v", err))
	}

	amount := decimal.RequireFromString(in.GetAmount())
	if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
		logger.Sugar().Errorw("validate", "Amount", in.GetAmount())
		return &ledger.CreateTransferResponse{}, status.Error(codes.InvalidArgument, "Amount is less than 0")
	}

	info, err := mledger.CreateTransfer(
		ctx,
		in.GetAppID(),
		in.GetUserID(),
		in.GetAccount(),
		in.GetAccountType(),
		in.GetVerificationCode(),
		in.GetTargetUserID(),
		in.GetAmount(),
		in.GetCoinTypeID())
	if err != nil {
		logger.Sugar().Errorw("CreateTransfer", "error", err)
		return &ledger.CreateTransferResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &ledger.CreateTransferResponse{
		Info: info,
	}, nil
}
