package respond

import (
	"errors"
	"net/http"
)

var errorResponseUnknown = errorResponse{
	Status:  http.StatusInternalServerError,
	Message: "unknown error",
}

// errorResponse contains both an error message and HTTP status code that provide
// enough information to respond with a meaningful error message and an appropriate
// 4XX/5XX status code to indicate the type of failure.
type errorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
}

// Status returns the HTTP status code you want to respond to the user with.
func (err errorResponse) StatusCode() int {
	return err.Status
}

// Error contains the error message you want to send back to the caller.
func (err errorResponse) Error() string {
	return err.Message
}

// ErrorWithStatus is a type of error that contains a Status() function which indicates
// the HTTP 4XX/5XX status code this type of failure should respond with.
type ErrorWithStatus interface {
	error
	Status() int
}

// ErrorWithStatusCode is a type of error that contains a StatusCode() function which indicates
// the HTTP 4XX/5XX status code this type of failure should respond with.
type ErrorWithStatusCode interface {
	error
	StatusCode() int
}

// ErrorWithCode is a type of error that contains a Code() function which indicates
// the HTTP 4XX/5XX status code this type of failure should respond with.
type ErrorWithCode interface {
	error
	Code() int
}

// toErrorWithStatus attempts to unwrap the given error, looking for Status(), StatusCode(), or
// Code() functions to extract a 4XX/5XX response, returning a strongly typed ErrorWithStatusCode that
// contains the HTTP status code and error message you can respond with.
func toErrorResponse(err error) errorResponse {
	if err == nil {
		return errorResponseUnknown
	}

	var errStatus ErrorWithStatus
	if errors.As(err, &errStatus) {
		return errorResponse{
			Status:  errStatus.Status(),
			Message: errStatus.Error(),
		}
	}

	var errStatusCode ErrorWithStatusCode
	if errors.As(err, &errStatusCode) {
		return errorResponse{
			Status:  errStatusCode.StatusCode(),
			Message: errStatusCode.Error(),
		}
	}

	var errCode ErrorWithCode
	if errors.As(err, &errCode) {
		return errorResponse{
			Status:  errCode.Code(),
			Message: errCode.Error(),
		}
	}

	return errorResponse{
		Status:  http.StatusInternalServerError,
		Message: err.Error(),
	}
}
