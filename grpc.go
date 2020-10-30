package srvgrpc

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// GRPCConfiguration abstracts the implementation for the GRPCService configuration.
type GRPCConfiguration interface {
	// BindAddr returns the bind address for the GRPC server when `GRPCService` is starting. It will be called only
	// when the `Start` of the `GRPCService` is called.
	BindAddr() (string, error)
}

// GRPCSetupFunc receives the `grpc.Server` instance and should register it on target package.
type GRPCSetupFunc func(*grpc.Server)

// GRPCService is the `services.Service` implementation for GRPC services.
type GRPCService struct {
	name   string
	config GRPCConfiguration

	serverCh      chan bool
	ctxLock       sync.Mutex
	ctx           context.Context
	cancelFunc    context.CancelFunc
	serverOptions []grpc.ServerOption
	setupFunc     GRPCSetupFunc
}

// NewGRPCService returns a new instance of the `GRPCService`.
//
// Arguments:
// - `name` identifies the service, required by the `github.com/setare/services.Service` interface.
// - `config` will provide additional configuration for the grpc.Server initialization. It is a `GRPCConfiguration`
//   interface implementation.
// - `setupFunc` is the callback responsible for registering the implementation to the `*grpc.Server` that is passed as
//   argument.
func NewGRPCService(name string, config GRPCConfiguration, setupFunc GRPCSetupFunc) *GRPCService {
	return &GRPCService{
		name:      name,
		config:    config,
		setupFunc: setupFunc,
	}
}

// ServerOptions set the `grpc.ServerOption` array that is passed when the service starts the `*grpc.Server`.
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
	service.ctxLock.Lock()
	defer func() {
		service.ctx = nil
		service.cancelFunc = nil
		service.ctxLock.Unlock()
	}()
	if service.cancelFunc != nil {
		service.cancelFunc()
	}
	<-service.serverCh // Waits for the shutdown
	return nil
}

// StartWithContext start the service in a blocking way. This is cancellable, so the context received can be
// cancelled at any moment. If your start implementation is not cancellable, you should implement `Startable`
// instead.
//
// If the service is successfully started, `nil` should be returned. Otherwise, an error must be returned.
func (service *GRPCService) StartWithContext(ctx context.Context) error {
	service.ctxLock.Lock()
	service.ctx, service.cancelFunc = context.WithCancel(ctx)
	service.ctxLock.Unlock()

	bindAddr, err := service.config.BindAddr()
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(service.serverOptions...)
	service.setupFunc(grpcServer)

	// Channel that will receive th error if the `grpServer.Serve` fails. This approach
	errCh := make(chan error)
	go func() {
		service.serverCh = make(chan bool)
		// Starts the GRPC server on the listener.
		// errCh will receive any error, since this is starting on a goroutine.
		err := grpcServer.Serve(lis)
		if err != http.ErrServerClosed && err != nil {
			errCh <- err
		}
		// The `Stop` method blacks using this channel for making sure the shutdown process has been finished before
		// unblocking.
		close(service.serverCh)
	}()

	go func() {
		// Wait for the context to be done.
		select {
		case <-ctx.Done():
		case <-service.ctx.Done():
		}
		grpcServer.GracefulStop()
	}()

	// Waits a second for the grpcServer to start.
	select {
	case err := <-errCh:
		return err
	case <-time.After(time.Millisecond * 100):
		// After 100 milliseconds and it didn't failed, it should be fine.
		return nil
	}
}
