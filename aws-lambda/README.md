## Usages:
### function-url-router
```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	router "github.com/rthing31/go/aws-lambda/function-url-router"
)

func main() {
	r := router.NewRouter()

	// Add middleware
	r.UsePre(router.MiddlewareFunc(loggerMiddleware))
	r.UsePost(router.MiddlewareFunc(headerMiddleware))

	// Add routes
	r.AddRoute(router.MethodGet, "/hello", router.HandlerFunc(helloHandler))
	r.AddRoute(router.MethodPost, "/echo", router.HandlerFunc(echoHandler))

	// Set custom Not Found handler
	r.SetNotFoundHandler(router.HandlerFunc(customNotFoundHandler))

	// For AWS Lambda
	lambda.Start(r.HandleRequest)
}

func loggerMiddleware(ctx context.Context, request events.LambdaFunctionURLRequest, next router.Handler) (events.LambdaFunctionURLResponse, error) {
	log.Printf("Request: %s %s", request.RequestContext.HTTP.Method, request.RequestContext.HTTP.Path)
	return next.HandleRequest(ctx, request)
}

func headerMiddleware(ctx context.Context, request events.LambdaFunctionURLRequest, next router.Handler) (events.LambdaFunctionURLResponse, error) {
	resp, err := next.HandleRequest(ctx, request)
	if err == nil {
		if resp.Headers == nil {
			resp.Headers = make(map[string]string)
		}
		resp.Headers["X-Custom-Header"] = "SomeValue"
	}
	return resp, err
}

func helloHandler(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusOK,
		Body:       "Hello, World!",
	}, nil
}

func echoHandler(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusOK,
		Body:       request.Body,
	}, nil
}

func customNotFoundHandler(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusNotFound,
		Body:       fmt.Sprintf("Custom 404 Not Found: %s", request.RequestContext.HTTP.Path),
	}, nil
}
```