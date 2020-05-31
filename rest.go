package srvgrpc

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

type RESTSetupFunc func(context.Context, *runtime.ServeMux, ...grpc.DialOption) error

type RESTService struct {
	serverCh      chan bool
	ctxLock       sync.Mutex
	ctx           context.Context
	cancelFunc    context.CancelFunc
	bindAddr      string
	name          string
	withUserAgent string
	withInsecure  bool
	dialOpts      []grpc.DialOption
	setupFunc     RESTSetupFunc
}

func NewRESTService(bindAddr, name string, setupFunc RESTSetupFunc) *RESTService {
	return &RESTService{
		bindAddr:  bindAddr,
		name:      name,
		setupFunc: setupFunc,
	}
}

func (service *RESTService) WithInsecure(value bool) *RESTService {
	service.withInsecure = value
	return service
}

func (service *RESTService) WithUserAgent(value string) *RESTService {
	service.withUserAgent = value
	return service
}

func (service *RESTService) DialOpts(options []grpc.DialOption) *RESTService {
	service.dialOpts = options
	return service
}

// Name will return a human identifiable name for this service. Ex: Postgresql Connection.
func (service *RESTService) Name() string {
	return service.name
}

// Stop will stop this service.
//
// For most implementations it will be blocking and should return only when the service finishes stopping.
//
// If the service is successfully stopped, `nil` should be returned. Otherwise, an error must be returned.
func (service *RESTService) Stop() error {
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
func (service *RESTService) StartWithContext(ctx context.Context) error {
	service.ctxLock.Lock()
	service.ctx, service.cancelFunc = context.WithCancel(ctx)
	service.ctxLock.Unlock()

	mux := runtime.NewServeMux()
	var opts []grpc.DialOption
	if len(service.dialOpts) > 0 {
		opts = service.dialOpts
	} else {
		opts = make([]grpc.DialOption, 0, 1)
	}

	if service.withInsecure {
		opts = append(opts, grpc.WithInsecure())
	}
	if service.withUserAgent != "" {
		opts = append(opts, grpc.WithUserAgent(service.withUserAgent))
	}
	err := service.setupFunc(ctx, mux, opts...)
	if err != nil {
		return err
	}

	httpServer := http.Server{
		Handler: mux,
	}

	errCh := make(chan error)
	go func() {
		service.serverCh = make(chan bool)
		// Starts the GRPC server on the listener.
		// errCh will receive any error, since this is starting on a goroutine.
		errCh <- httpServer.ListenAndServe()
		defer close(service.serverCh)
	}()

	go func() {
		select {
		case <-ctx.Done(): // Wait for the context to be done.
		case <-service.ctx.Done():
		}

		// Tries to shutdown for 20 seconds
		shutdownCtx, cancelFnc := context.WithTimeout(context.Background(), time.Second*20)
		defer cancelFnc()

		// Tries to gracefully shutdown the httpServer.
		httpServer.Shutdown(shutdownCtx)
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
