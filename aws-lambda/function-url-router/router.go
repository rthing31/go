package router

import (
	"context"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// Handler defines the interface for custom handlers
type Handler interface {
	HandleRequest(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error)
}

// HandlerFunc is a function type that implements the Handler interface
type HandlerFunc func(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error)

// HandleRequest calls f(ctx, request)
func (f HandlerFunc) HandleRequest(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return f(ctx, request)
}

// Middleware defines the interface for middleware
type Middleware interface {
	Process(ctx context.Context, request events.LambdaFunctionURLRequest, next Handler) (events.LambdaFunctionURLResponse, error)
}

// MiddlewareFunc is a function type that implements the Middleware interface
type MiddlewareFunc func(ctx context.Context, request events.LambdaFunctionURLRequest, next Handler) (events.LambdaFunctionURLResponse, error)

// Process calls f(ctx, request, next)
func (f MiddlewareFunc) Process(ctx context.Context, request events.LambdaFunctionURLRequest, next Handler) (events.LambdaFunctionURLResponse, error) {
	return f(ctx, request, next)
}

// Router is the main structure for routing
type Router struct {
	routes                  map[string]map[string]Handler
	preMiddleware           []Middleware
	postMiddleware          []Middleware
	notFoundHandler         Handler
	methodNotAllowedHandler Handler
}

// NewRouter initializes a new Router instance
func NewRouter() *Router {
	return &Router{
		routes:                  make(map[string]map[string]Handler),
		preMiddleware:           []Middleware{},
		postMiddleware:          []Middleware{},
		notFoundHandler:         HandlerFunc(defaultNotFoundHandler),
		methodNotAllowedHandler: HandlerFunc(defaultMethodNotAllowedHandler),
	}
}

// AddRoute registers a new route with a handler
func (r *Router) AddRoute(method, path string, handler Handler) {
	if r.routes[path] == nil {
		r.routes[path] = make(map[string]Handler)
	}
	r.routes[path][strings.ToUpper(method)] = handler
}

// UsePre adds a pre-route middleware to the router
func (r *Router) UsePre(mw Middleware) {
	r.preMiddleware = append(r.preMiddleware, mw)
}

// UsePost adds a post-route middleware to the router
func (r *Router) UsePost(mw Middleware) {
	r.postMiddleware = append(r.postMiddleware, mw)
}

// SetNotFoundHandler sets a custom handler for 404 Not Found responses
func (r *Router) SetNotFoundHandler(handler Handler) {
	r.notFoundHandler = handler
}

// SetMethodNotAllowedHandler sets a custom handler for 405 Method Not Allowed responses
func (r *Router) SetMethodNotAllowedHandler(handler Handler) {
	r.methodNotAllowedHandler = handler
}

// HandleRequest routes the incoming request to the appropriate handler
func (r *Router) HandleRequest(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	handler := r.createHandlerChain(r.preMiddleware, HandlerFunc(r.handleRouteRequest))
	handler = r.createHandlerChain(r.postMiddleware, handler)
	return handler.HandleRequest(ctx, request)
}

// handleRouteRequest routes the request to the appropriate handler
func (r *Router) handleRouteRequest(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	if methodHandlers, ok := r.routes[request.RequestContext.HTTP.Path]; ok {
		if handler, ok := methodHandlers[request.RequestContext.HTTP.Method]; ok {
			return handler.HandleRequest(ctx, request)
		}
		return r.methodNotAllowedHandler.HandleRequest(ctx, request)
	}
	return r.notFoundHandler.HandleRequest(ctx, request)
}

// createHandlerChain creates a chain of middleware and handlers
func (r *Router) createHandlerChain(middleware []Middleware, finalHandler Handler) Handler {
	if len(middleware) == 0 {
		return finalHandler
	}

	return HandlerFunc(func(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
		return middleware[0].Process(ctx, request, r.createHandlerChain(middleware[1:], finalHandler))
	})
}

func defaultNotFoundHandler(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{
		StatusCode: StatusNotFound,
		Body:       "Not Found",
	}, nil
}

func defaultMethodNotAllowedHandler(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{
		StatusCode: StatusMethodNotAllowed,
		Body:       "Method Not Allowed",
	}, nil
}
