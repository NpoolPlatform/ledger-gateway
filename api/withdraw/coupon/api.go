package coupon

import (
	"context"

	coupon1 "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw/coupon"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Server struct {
	coupon1.UnimplementedGatewayServer
}

func Register(server grpc.ServiceRegistrar) {
	coupon1.RegisterGatewayServer(server, &Server{})
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	if err := coupon1.RegisterGatewayHandlerFromEndpoint(context.Background(), mux, endpoint, opts); err != nil {
		return err
	}
	return nil
}
