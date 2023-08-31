package withdraw

import (
	"context"

	withdraw "github.com/NpoolPlatform/message/npool/ledger/gw/v1/withdraw"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Server struct {
	withdraw.UnimplementedGatewayServer
}

func Register(server grpc.ServiceRegistrar) {
	withdraw.RegisterGatewayServer(server, &Server{})
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	if err := withdraw.RegisterGatewayHandlerFromEndpoint(context.Background(), mux, endpoint, opts); err != nil {
		return err
	}
	return nil
}
