package router

// HTTP methods
const (
	MethodGet     = "GET"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodPatch   = "PATCH"
	MethodOptions = "OPTIONS"
	MethodHead    = "HEAD"
	MethodConnect = "CONNECT"
	MethodTrace   = "TRACE"
)

// HTTP status codes for default handlers
const (
	StatusNotFound         = 404
	StatusMethodNotAllowed = 405
)
