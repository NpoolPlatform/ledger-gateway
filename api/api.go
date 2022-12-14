package api

import (
	"context"

	ledger "github.com/NpoolPlatform/message/npool/ledger/gw/v1"

	ledger1 "github.com/NpoolPlatform/ledger-gateway/api/ledger"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Server struct {
	ledger.UnimplementedGatewayServer
}

func Register(server grpc.ServiceRegistrar) {
	ledger.RegisterGatewayServer(server, &Server{})
	ledger1.Register(server)
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	if err := ledger.RegisterGatewayHandlerFromEndpoint(context.Background(), mux, endpoint, opts); err != nil {
		return err
	}
	if err := ledger1.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	return nil
}
