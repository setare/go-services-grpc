package srvgrpc

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

type RESTSetupFunc func(context.Context, *runtime.ServeMux, ...grpc.DialOption) error

type RESTService struct {
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
	panic("not implemented") // TODO: Implement
}

// StartWithContext start the service in a blocking way. This is cancellable, so the context received can be
// cancelled at any moment. If your start implementation is not cancellable, you should implement `Startable`
// instead.
//
// If the service is successfully started, `nil` should be returned. Otherwise, an error must be returned.
func (service *RESTService) StartWithContext(ctx context.Context) error {
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

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	return http.ListenAndServe(service.bindAddr, mux)

}
