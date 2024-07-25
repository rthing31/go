package router

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

type Handler interface {
	ServeHTTP(context.Context, events.LambdaFunctionURLRequest) Response
}

type HandlerFunc func(context.Context, events.LambdaFunctionURLRequest) Response

func (f HandlerFunc) ServeHTTP(ctx context.Context, req events.LambdaFunctionURLRequest) Response {
	return f(ctx, req)
}

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       interface{}       `json:"body"`
}

type MiddlewareFunc func(Handler) Handler

type MiddlewareConfig struct {
	ExcludedRoutes  []string
	ExcludedMethods []string
	ExcludedHeaders map[string]string
}

type Middleware struct {
	Func   MiddlewareFunc
	Config MiddlewareConfig
}

type Router struct {
	routes                  map[string]map[string]Handler
	preMiddleware           []Middleware
	postMiddleware          []Middleware
	notFoundHandler         Handler
	methodNotAllowedHandler Handler
	panicHandler            func(context.Context, events.LambdaFunctionURLRequest) Response
	stripTrailingSlash      bool
	logger                  *log.Logger
}

func NewRouter(logger *log.Logger) *Router {
	if logger == nil {
		logger = log.New(os.Stdout, "ROUTER: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
	r := &Router{
		routes:             make(map[string]map[string]Handler),
		stripTrailingSlash: true,
		logger:             logger,
	}
	r.notFoundHandler = HandlerFunc(defaultNotFoundHandler)
	r.methodNotAllowedHandler = HandlerFunc(defaultMethodNotAllowedHandler)
	r.panicHandler = defaultPanicHandler
	return r
}

func (r *Router) AddRoute(method, path string, handler Handler) {
	if r.routes[path] == nil {
		r.routes[path] = make(map[string]Handler)
	}
	r.routes[path][method] = handler
}

func (r *Router) UsePre(mw MiddlewareFunc, config MiddlewareConfig) {
	r.preMiddleware = append(r.preMiddleware, Middleware{Func: mw, Config: config})
}

func (r *Router) UsePost(mw MiddlewareFunc, config MiddlewareConfig) {
	r.postMiddleware = append(r.postMiddleware, Middleware{Func: mw, Config: config})
}

func (r *Router) SetNotFoundHandler(handler Handler) {
	r.notFoundHandler = handler
}

func (r *Router) SetMethodNotAllowedHandler(handler Handler) {
	r.methodNotAllowedHandler = handler
}

func (r *Router) SetPanicHandler(handler func(context.Context, events.LambdaFunctionURLRequest) Response) {
	r.panicHandler = handler
}

func (r *Router) SetStripTrailingSlash(strip bool) {
	r.stripTrailingSlash = strip
}

func (r *Router) HandleRequest(ctx context.Context, req events.LambdaFunctionURLRequest) Response {
	startTime := time.Now()
	var resp Response
	var err error

	defer func() {
		duration := time.Since(startTime)
		if e := recover(); e != nil {
			err = fmt.Errorf("panic: %v", e)
			resp = r.panicHandler(ctx, req)
		}
		r.logRequestCompletion(req, resp, duration, err)
	}()

	path := req.RequestContext.HTTP.Path
	method := req.RequestContext.HTTP.Method

	if r.stripTrailingSlash {
		path = strings.TrimRight(path, "/")
	}

	if handlers, ok := r.routes[path]; ok {
		if handler, ok := handlers[method]; ok {
			handler = r.applyMiddleware(handler)
			resp = handler.ServeHTTP(ctx, req)
			return resp
		}
		resp = r.methodNotAllowedHandler.ServeHTTP(ctx, req)
		return resp
	}
	resp = r.notFoundHandler.ServeHTTP(ctx, req)
	return resp
}

func (r *Router) applyMiddleware(handler Handler) Handler {
	for i := len(r.postMiddleware) - 1; i >= 0; i-- {
		mw := r.postMiddleware[i]
		handler = mw.Func(handler)
	}

	for i := len(r.preMiddleware) - 1; i >= 0; i-- {
		mw := r.preMiddleware[i]
		handler = mw.Func(handler)
	}

	return handler
}

func (r *Router) logRequestCompletion(req events.LambdaFunctionURLRequest, resp Response, duration time.Duration, err error) {
	logEntry := fmt.Sprintf(
		"Request completed: method=%s path=%s status=%d duration=%v",
		req.RequestContext.HTTP.Method,
		req.RequestContext.HTTP.Path,
		resp.StatusCode,
		duration,
	)

	if err != nil {
		logEntry += fmt.Sprintf(" error=%v", err)
	}

	r.logger.Println(logEntry)
}

func defaultNotFoundHandler(ctx context.Context, req events.LambdaFunctionURLRequest) Response {
	return Response{
		StatusCode: http.StatusNotFound,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       map[string]string{"error": "Not Found"},
	}
}

func defaultMethodNotAllowedHandler(ctx context.Context, req events.LambdaFunctionURLRequest) Response {
	return Response{
		StatusCode: http.StatusMethodNotAllowed,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       map[string]string{"error": "Method Not Allowed"},
	}
}

func defaultPanicHandler(ctx context.Context, req events.LambdaFunctionURLRequest) Response {
	return Response{
		StatusCode: http.StatusInternalServerError,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       map[string]string{"error": "Internal Server Error"},
	}
}
