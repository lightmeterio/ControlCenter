// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package httperror

// HTTPError represents an HTTP error with HTTP status code and error message
type XHTTPError interface {
	error
	// StatusCode returns the HTTP status code of the error
	StatusCode() int
	JSON() bool
	ErrObject() error
}

type HttpCodeError struct {
	statusCode int
	Err        error
}

// NewHTTPStatusCodeError creates a new HttpError instance.
// to generate the message based on the status code.
func NewHTTPStatusCodeError(code int, err error) XHTTPError {
	return &HttpCodeError{statusCode: code, Err: err}
}

func (e *HttpCodeError) Error() string {
	return e.Err.Error()
}

func (e *HttpCodeError) ErrObject() error {
	return e.Err
}

// StatusCode returns the HTTP status code.
func (e *HttpCodeError) StatusCode() int {
	return e.statusCode
}

func (e *HttpCodeError) JSON() bool {
	return false
}

type HttpCodeJsonError struct {
	statusCode int
	Err        error
}

// NewHTTPStatusCodeError creates a new HttpError instance.
// to generate the message based on the status code.
func NewHttpCodeJsonError(code int, err error) XHTTPError {
	return &HttpCodeJsonError{statusCode: code, Err: err}
}

func (e *HttpCodeJsonError) Error() string {
	return e.Err.Error()
}

func (e *HttpCodeJsonError) ErrObject() error {
	return e.Err
}

// StatusCode returns the HTTP status code.
func (e *HttpCodeJsonError) StatusCode() int {
	return e.statusCode
}

func (e *HttpCodeJsonError) JSON() bool {
	return true
}
