package srvgrpc

import (
	"context"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
)

type GRPCSetupFunc func(*grpc.Server)

type GRPCService struct {
	bindAddr      string
	name          string
	serverOptions []grpc.ServerOption
	setupFunc     GRPCSetupFunc
}

func NewGRPCService(bindAddr, name string, setupFunc GRPCSetupFunc) *GRPCService {
	return &GRPCService{
		bindAddr:  bindAddr,
		name:      name,
		setupFunc: setupFunc,
	}
}

func (service *GRPCService) ServerOptions(options ...grpc.ServerOption) *GRPCService {
	service.serverOptions = options
	return service
}

// Name will return a human identifiable name for this service. Ex: Postgresql Connection.
func (service *GRPCService) Name() string {
	return service.name
}

// Stop will stop this service.
//
// For most implementations it will be blocking and should return only when the service finishes stopping.
//
// If the service is successfully stopped, `nil` should be returned. Otherwise, an error must be returned.
func (service *GRPCService) Stop() error {
	panic("not implemented") // TODO: Implement
}

// StartWithContext start the service in a blocking way. This is cancellable, so the context received can be
// cancelled at any moment. If your start implementation is not cancellable, you should implement `Startable`
// instead.
//
// If the service is successfully started, `nil` should be returned. Otherwise, an error must be returned.
func (service *GRPCService) StartWithContext(ctx context.Context) error {
	lis, err := net.Listen("tcp", service.bindAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(service.serverOptions...)
	service.setupFunc(grpcServer)

	errCh := make(chan error)
	go func() {
		errCh <- grpcServer.Serve(lis)
	}()

	go func() {
		select {
		case <-ctx.Done():
			grpcServer.GracefulStop()
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(time.Second):
		// After one second and it didn't failed, it should be fine.
		return nil
	}
}
