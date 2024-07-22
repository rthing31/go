package router

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

// Handler defines the interface for custom handlers
type Handler interface {
	HandleRequest(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error)
}

// Router is the main structure for routing
type Router struct {
	routes map[string]map[string]Handler
}

// NewRouter initializes a new Router instance
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]map[string]Handler),
	}
}

// AddRoute registers a new route with a handler
func (r *Router) AddRoute(method, path string, handler Handler) {
	if r.routes[path] == nil {
		r.routes[path] = make(map[string]Handler)
	}
	r.routes[path][method] = handler
}

// HandleRequest routes the incoming request to the appropriate handler
func (r *Router) HandleRequest(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	if methodHandlers, ok := r.routes[request.RawPath]; ok {
		if handler, ok := methodHandlers[request.RequestContext.HTTP.Method]; ok {
			return handler.HandleRequest(ctx, request)
		}
		return events.LambdaFunctionURLResponse{
			StatusCode: 405,
			Body:       "Method Not Allowed",
		}, nil
	}
	return events.LambdaFunctionURLResponse{
		StatusCode: 404,
		Body:       "Not Found",
	}, nil
}
