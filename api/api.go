package api

import (
	"context"

	ledger "github.com/NpoolPlatform/message/npool/ledger/gw/v1"

	ledger1 "github.com/NpoolPlatform/ledger-gateway/api/ledger"
	deposit "github.com/NpoolPlatform/ledger-gateway/api/ledger/deposit"
	profit "github.com/NpoolPlatform/ledger-gateway/api/ledger/profit"
	statement "github.com/NpoolPlatform/ledger-gateway/api/ledger/statement"
	transfer "github.com/NpoolPlatform/ledger-gateway/api/ledger/transfer"
	withdraw "github.com/NpoolPlatform/ledger-gateway/api/withdraw"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Server struct {
	ledger.UnimplementedGatewayServer
}

func Register(server grpc.ServiceRegistrar) {
	ledger.RegisterGatewayServer(server, &Server{})
	ledger1.Register(server)
	deposit.Register(server)
	transfer.Register(server)
	profit.Register(server)
	statement.Register(server)
	withdraw.Register(server)
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	if err := ledger.RegisterGatewayHandlerFromEndpoint(context.Background(), mux, endpoint, opts); err != nil {
		return err
	}
	if err := ledger1.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := deposit.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := transfer.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := profit.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := statement.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	if err := withdraw.RegisterGateway(mux, endpoint, opts); err != nil {
		return err
	}
	return nil
}
