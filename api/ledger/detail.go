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
