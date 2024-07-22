package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	router "github.com/rthing31/go/aws-lambda/function-url-router"
)

func main() {
	r := router.NewRouter()

	// Register custom handler for different endpoints and methods
	r.AddRoute(router.MethodGet, "/custom", CustomHandler{})

	// Start the Lambda handler
	lambda.Start(r.HandleRequest)
}

// CustomHandler is a user-defined handler
type CustomHandler struct{}

// HandleRequest handles the request
func (h CustomHandler) HandleRequest(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{
		StatusCode: 200,
		Body:       fmt.Sprintf("Handled custom request at %s", request.RawPath),
	}, nil
}
