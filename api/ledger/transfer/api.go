package transfer

import (
	"context"

	transfer "github.com/NpoolPlatform/message/npool/ledger/gw/v1/ledger/transfer"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Server struct {
	transfer.UnimplementedGatewayServer
}

func Register(server grpc.ServiceRegistrar) {
	transfer.RegisterGatewayServer(server, &Server{})
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	if err := transfer.RegisterGatewayHandlerFromEndpoint(context.Background(), mux, endpoint, opts); err != nil {
		return err
	}
	return nil
}
