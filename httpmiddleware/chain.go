package httpmiddleware

import (
	"log"
	"net/http"
)

type CustomHTTPHandlerInterface interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request) error
}

type CustomHTTPHandler func(w http.ResponseWriter, r *http.Request) error

func (f CustomHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	return f(w, r)
}

type Middleware func(CustomHTTPHandler) CustomHTTPHandler

type Chain struct {
	middleware []Middleware
}

func New(middleware ...Middleware) Chain {
	return Chain{middleware: middleware}
}

func (c Chain) WithEndpoint(endpoint CustomHTTPHandlerInterface) http.Handler {
	if endpoint == nil {
		panic("endpoint is nil")
	}

	for i := range c.middleware {
		if c.middleware == nil {
			panic("middleware is nil")
		}

		endpoint = c.middleware[len(c.middleware)-1-i](endpoint.ServeHTTP)
	}

	return wrapWithErrorHandler(endpoint)
}

func (c Chain) WithError(endpoint CustomHTTPHandlerInterface) http.Handler {
	if endpoint == nil {
		panic("endpoint is nil")
	}

	return wrapWithErrorHandler(endpoint)
}

func wrapWithErrorHandler(endpoint CustomHTTPHandlerInterface) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		err := endpoint.ServeHTTP(w, r)
		switch errType := err.(type) {
		case *HttpCodeError:
			if errType.statusCode >= 500 {
				log.Println(err)
				w.WriteHeader(errType.statusCode)
				return
			}
			http.Error(w, errType.err.Error(), errType.statusCode)
		case error:
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}

// HTTPError represents an HTTP error with HTTP status code and error message
type XHTTPError interface {
	error
	// StatusCode returns the HTTP status code of the error
	StatusCode() int
}

type HttpCodeError struct {
	statusCode int
	err        error
}

// NewHTTPStatusCodeError creates a new HttpError instance.
// to generate the message based on the status code.
func NewHTTPStatusCodeError(code int, err error) XHTTPError {
	return &HttpCodeError{statusCode: code, err: err}
}

func (e *HttpCodeError) Error() string {
	return http.StatusText(e.statusCode)
}

// StatusCode returns the HTTP status code.
func (e *HttpCodeError) StatusCode() int {
	return e.statusCode
}
