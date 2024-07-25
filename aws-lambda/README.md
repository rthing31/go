## Usages:
### function-url-router
```go
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	router "github.com/rthing31/go/aws-lambda/function-url-router"
)

const (
	AuthToken = "your-secret-token-here"
)

func main() {
	logger := log.New(os.Stdout, "MAIN: ", log.Ldate|log.Ltime|log.Lshortfile)
	r := router.NewRouter(logger)

	r.SetStripTrailingSlash(true)

	r.AddRoute(http.MethodGet, "/hello", router.HandlerFunc(helloHandler))
	r.AddRoute(http.MethodPost, "/echo", router.HandlerFunc(echoHandler))

	r.UsePre(loggingMiddleware, router.MiddlewareConfig{
		ExcludedRoutes:  []string{"/health"},
		ExcludedMethods: []string{http.MethodOptions},
		ExcludedHeaders: map[string]string{"X-Skip-Logging": "true"},
	})

	r.UsePre(authMiddleware, router.MiddlewareConfig{
		ExcludedRoutes: []string{"/health", "/public"},
	})

	r.UsePost(headerMiddleware, router.MiddlewareConfig{})

	if isLambda() {
		lambda.Start(func(ctx context.Context, req events.LambdaFunctionURLRequest) (router.Response, error) {
			return r.HandleRequest(ctx, req), nil
		})
	} else {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		addr := ":" + port
		logger.Printf("Starting server on %s", addr)
		if err := router.RunLocalServer(r, addr, logger); err != nil {
			logger.Fatalf("Server error: %v", err)
		}
	}
}

func isLambda() bool {
	return os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != ""
}

func helloHandler(ctx context.Context, req events.LambdaFunctionURLRequest) router.Response {
	return router.Response{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       map[string]string{"message": "Hello, World!"},
	}
}

func echoHandler(ctx context.Context, req events.LambdaFunctionURLRequest) router.Response {
	var body interface{}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return router.Response{
			StatusCode: http.StatusBadRequest,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       map[string]string{"error": "Invalid JSON"},
		}
	}
	return router.Response{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
}

func loggingMiddleware(next router.Handler) router.Handler {
	return router.HandlerFunc(func(ctx context.Context, req events.LambdaFunctionURLRequest) router.Response {
		log.Printf("Request: %s %s", req.RequestContext.HTTP.Method, req.RequestContext.HTTP.Path)
		return next.ServeHTTP(ctx, req)
	})
}

func authMiddleware(next router.Handler) router.Handler {
	return router.HandlerFunc(func(ctx context.Context, req events.LambdaFunctionURLRequest) router.Response {
		token := req.Headers["authorization"]

		if token != AuthToken {
			return router.Response{
				StatusCode: http.StatusUnauthorized,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       map[string]string{"error": "Unauthorized"},
			}
		}

		return next.ServeHTTP(ctx, req)
	})
}

func headerMiddleware(next router.Handler) router.Handler {
	return router.HandlerFunc(func(ctx context.Context, req events.LambdaFunctionURLRequest) router.Response {
		resp := next.ServeHTTP(ctx, req)
		if resp.Headers == nil {
			resp.Headers = make(map[string]string)
		}
		resp.Headers["X-Custom-Header"] = "SomeValue"
		return resp
	})
}
```