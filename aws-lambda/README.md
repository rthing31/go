## Usages:
### function-url-router
```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	router "github.com/rthing31/go/aws-lambda/function-url-router"
)

func main() {
	r := router.NewRouter()

	// Add middleware
	r.UsePre(loggingMiddleware())
	r.UsePre(authMiddleware())

	// Add routes
	r.AddRoute(router.MethodGet, "/hello", router.HandlerFunc(helloHandler))
	r.AddRoute(router.MethodPost, "/users", router.HandlerFunc(createUserHandler))

	// Set custom not found handler
	r.SetNotFoundHandler(router.HandlerFunc(customNotFoundHandler))

	// Start the Lambda handler
	lambda.Start(r.HandleRequest)
}

func loggingMiddleware() router.Middleware {
	return router.NewMiddleware(
		func(ctx context.Context, request events.LambdaFunctionURLRequest, next router.Handler) (events.LambdaFunctionURLResponse, error) {
			log.Printf("Request: %s %s", request.RequestContext.HTTP.Method, request.RequestContext.HTTP.Path)
			return next.HandleRequest(ctx, request)
		},
		nil, // No exclusions
	)
}

func authMiddleware() router.Middleware {
	return router.NewMiddleware(
		func(ctx context.Context, request events.LambdaFunctionURLRequest, next router.Handler) (events.LambdaFunctionURLResponse, error) {
			// Perform authentication logic here
			// For demonstration, we'll just check for a token in the header
			if token, ok := request.Headers["authorization"]; !ok || token != "Bearer valid-token" {
				return events.LambdaFunctionURLResponse{
					StatusCode: router.StatusUnauthorized,
					Body:       "Unauthorized",
				}, nil
			}
			return next.HandleRequest(ctx, request)
		},
		&router.MiddlewareConfig{
			ExcludedEndpoints: []string{"/hello"},
			ExcludedMethods:   []string{router.MethodOptions},
		},
	)
}

func helloHandler(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{
		StatusCode: router.StatusOK,
		Body:       "Hello, World!",
	}, nil
}

func createUserHandler(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	// Process user creation logic here
	return events.LambdaFunctionURLResponse{
		StatusCode: router.StatusCreated,
		Body:       "User created successfully",
	}, nil
}

func customNotFoundHandler(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{
		StatusCode: router.StatusNotFound,
		Body:       fmt.Sprintf("Custom Not Found: %s", request.RequestContext.HTTP.Path),
	}, nil
}
```