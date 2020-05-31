package srvgrpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc"
)

type GRPCSetupFunc func(*grpc.Server)

type GRPCService struct {
	serverCh      chan bool
	ctxLock       sync.Mutex
	ctx           context.Context
	cancelFunc    context.CancelFunc
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

	lis, err := net.Listen("tcp", service.bindAddr)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(service.serverOptions...)
	service.setupFunc(grpcServer)

	errCh := make(chan error)
	go func() {
		service.serverCh = make(chan bool)
		// Starts the GRPC server on the listener.
		// errCh will receive any error, since this is starting on a goroutine.
		err := grpcServer.Serve(lis)
		fmt.Println(err)
		if err != http.ErrServerClosed && err != nil {
			errCh <- err
		}
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
	case <-time.After(time.Second):
		// After one second and it didn't failed, it should be fine.
		return nil
	}
}
