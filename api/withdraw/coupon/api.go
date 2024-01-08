package coupon

import (
	"context"

	couponwithdraw "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw/coupon"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Server struct {
	couponwithdraw.UnimplementedGatewayServer
}

func Register(server grpc.ServiceRegistrar) {
	couponwithdraw.RegisterGatewayServer(server, &Server{})
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	if err := couponwithdraw.RegisterGatewayHandlerFromEndpoint(context.Background(), mux, endpoint, opts); err != nil {
		return err
	}
	return nil
}
