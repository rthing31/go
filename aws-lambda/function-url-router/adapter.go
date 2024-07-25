package router

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

type LambdaAdapter struct {
	router *Router
	logger *log.Logger
}

func NewLambdaAdapter(router *Router, logger *log.Logger) *LambdaAdapter {
	if logger == nil {
		logger = log.New(os.Stdout, "ADAPTER: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
	return &LambdaAdapter{router: router, logger: logger}
}

func (la *LambdaAdapter) ServeHTTP(r *http.Request) Response {
	lambdaReq := la.httpToLambdaRequest(r)
	return la.router.HandleRequest(r.Context(), lambdaReq)
}

func (la *LambdaAdapter) httpToLambdaRequest(r *http.Request) events.LambdaFunctionURLRequest {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		la.logger.Printf("Error reading request body: %v", err)
	}
	defer r.Body.Close()

	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[strings.ToLower(k)] = strings.Join(v, ",")
	}

	queryParams := make(map[string]string)
	for k, v := range r.URL.Query() {
		queryParams[k] = strings.Join(v, ",")
	}

	cookies := make([]string, 0)
	for _, cookie := range r.Cookies() {
		cookies = append(cookies, cookie.String())
	}

	now := time.Now()

	return events.LambdaFunctionURLRequest{
		Version:               "2.0",
		RawPath:               r.URL.Path,
		RawQueryString:        r.URL.RawQuery,
		Cookies:               cookies,
		Headers:               headers,
		QueryStringParameters: queryParams,
		RequestContext: events.LambdaFunctionURLRequestContext{
			AccountID:    "123456789012",
			RequestID:    "dummy-request-id",
			Authorizer:   nil,
			APIID:        "dummy-api-id",
			DomainName:   "dummy.lambda-url.us-east-1.on.aws",
			DomainPrefix: "dummy",
			Time:         now.Format(time.RFC3339),
			TimeEpoch:    now.UnixNano() / int64(time.Millisecond),
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{
				Method:    r.Method,
				Path:      r.URL.Path,
				Protocol:  r.Proto,
				SourceIP:  r.RemoteAddr,
				UserAgent: r.UserAgent(),
			},
		},
		Body:            string(body),
		IsBase64Encoded: false,
	}
}

func RunLocalServer(router *Router, addr string, logger *log.Logger) error {
	if logger == nil {
		logger = log.New(os.Stdout, "SERVER: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
	adapter := NewLambdaAdapter(router, logger)
	logger.Printf("Starting local server on %s", addr)
	return http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := adapter.ServeHTTP(r)
		for k, v := range resp.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(resp.StatusCode)
		if err := json.NewEncoder(w).Encode(resp.Body); err != nil {
			logger.Printf("Error encoding response body: %v", err)
		}
	}))
}
