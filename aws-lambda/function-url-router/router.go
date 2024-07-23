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

// MiddlewareConfig defines the configuration for middleware exclusions
type MiddlewareConfig struct {
	ExcludedEndpoints []string
	ExcludedMethods   []string
	ExcludedHeaders   map[string]string
}

// Middleware defines the interface for middleware
type Middleware interface {
	Process(ctx context.Context, request events.LambdaFunctionURLRequest, next Handler) (events.LambdaFunctionURLResponse, error)
	Config() *MiddlewareConfig
}

// MiddlewareFunc is a function type that implements the Middleware interface
type MiddlewareFunc struct {
	Func   func(ctx context.Context, request events.LambdaFunctionURLRequest, next Handler) (events.LambdaFunctionURLResponse, error)
	config *MiddlewareConfig
}

// Process calls f.Func(ctx, request, next)
func (f MiddlewareFunc) Process(ctx context.Context, request events.LambdaFunctionURLRequest, next Handler) (events.LambdaFunctionURLResponse, error) {
	return f.Func(ctx, request, next)
}

// Config returns the middleware configuration
func (f MiddlewareFunc) Config() *MiddlewareConfig {
	return f.config
}

// NewMiddleware creates a new MiddlewareFunc with the given function and config
func NewMiddleware(f func(ctx context.Context, request events.LambdaFunctionURLRequest, next Handler) (events.LambdaFunctionURLResponse, error), config *MiddlewareConfig) Middleware {
	return MiddlewareFunc{Func: f, config: config}
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
	handler := r.createHandlerChain(r.preMiddleware, HandlerFunc(r.handleRouteRequest), request)
	handler = r.createHandlerChain(r.postMiddleware, handler, request)
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

// createHandlerChain creates a chain of middleware and handlers, considering exclusions
func (r *Router) createHandlerChain(middleware []Middleware, finalHandler Handler, request events.LambdaFunctionURLRequest) Handler {
	if len(middleware) == 0 {
		return finalHandler
	}

	return HandlerFunc(func(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
		if r.shouldApplyMiddleware(middleware[0], req) {
			return middleware[0].Process(ctx, req, r.createHandlerChain(middleware[1:], finalHandler, req))
		}
		return r.createHandlerChain(middleware[1:], finalHandler, req).HandleRequest(ctx, req)
	})
}

// shouldApplyMiddleware checks if the middleware should be applied based on its configuration
func (r *Router) shouldApplyMiddleware(mw Middleware, request events.LambdaFunctionURLRequest) bool {
	config := mw.Config()
	if config == nil {
		return true
	}

	// Check excluded endpoints
	for _, endpoint := range config.ExcludedEndpoints {
		if request.RequestContext.HTTP.Path == endpoint {
			return false
		}
	}

	// Check excluded methods
	for _, method := range config.ExcludedMethods {
		if request.RequestContext.HTTP.Method == method {
			return false
		}
	}

	// Check excluded headers
	for header, value := range config.ExcludedHeaders {
		if headerValue, ok := request.Headers[header]; ok && headerValue == value {
			return false
		}
	}

	return true
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
