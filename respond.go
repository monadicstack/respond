package respond

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

func To(w http.ResponseWriter, req *http.Request) Responder {
	return Responder{writer: w, request: req}
}

type Responder struct {
	writer  http.ResponseWriter
	request *http.Request
}

// Reply lets you respond with the custom status code of your choice and a JSON-marshaled version of your value.
func (r Responder) Reply(status int, value interface{}, errs ...error) {
	if err := firstError(errs...); err != nil {
		r.Fail(err)
		return
	}
	writeJSON(r.writer, status, value)
}

// Ok writes a 200 style response to the caller by marshalling the given raw value. If
// you provided an error, we'll ignore the value and return the appropriate 4XX/5XX
// response instead.
func (r Responder) Ok(value interface{}, errs ...error) {
	r.Reply(http.StatusOK, value, errs...)
}

// Ok writes a 201 style response to the caller by marshalling the given raw value. If
// you provided an error, we'll ignore the value and return the appropriate 4XX/5XX
// response instead.
func (r Responder) Created(value interface{}, errs ...error) {
	r.Reply(http.StatusCreated, value, errs...)
}

// Accepted writes a 202 style response to the caller by marshalling the given raw value. If
// you provided an error, we'll ignore the value and return the appropriate 4XX/5XX
// response instead.
func (r Responder) Accepted(value interface{}, errs ...error) {
	r.Reply(http.StatusAccepted, value, errs...)
}

// NoContent writes a 204 style response to the caller. This will not write any bytes to the
// response other than the status code. If you provided an error, we'll ignore the 204 and
// return the appropriate 4XX/5XX response instead.
func (r Responder) NoContent(errs ...error) {
	if err := firstError(errs...); err != nil {
		r.Fail(err)
		return
	}
	r.writer.WriteHeader(http.StatusNoContent)
}

// Serve responds with some sort of file data in an inline fashion. This lets you deliver
// things like inline images or videos or any other content that you want your callers/clients
// to embed directly in the client. The file name in this case case is simply used to determine
// the proper Content-Type to include in the response.
//
// It will read your 'data' stream to completion but it will still be up to you to Close() it
// afterwards if need be.
func (r Responder) Serve(fileName string, data io.Reader) {
	r.writer.Header().Set("Content-Type", fileNameToContentType(fileName))
	r.writer.Header().Set("Content-Disposition", "inline")
	r.writer.WriteHeader(http.StatusOK)

	if data == nil {
		return
	}

	_, err := io.Copy(r.writer, data)
	if err != nil {
		http.Error(r.writer, err.Error(), http.StatusInternalServerError)
	}
}

// ServeBytes responds with some sort of file data in an inline fashion. This lets you deliver
// things like inline images or videos or any other content that you want your callers/clients
// to embed directly in the client. The file name in this case case is simply used to determine
// the proper Content-Type to include in the response.
func (r Responder) ServeBytes(fileName string, data []byte) {
	r.Serve(fileName, bytes.NewBuffer(data))
}

// Download delivers the file data to the client/caller in a way that indicates that it should
// be given a download prompt (if using a browser or some other UI-based client). The file name
// determines the Content-Type header we'll use in the response as well as be the default download
// name that the caller will be presented with in their client/browser.
//
// It will read your 'data' stream to completion but it will still be up to you to Close() it
// afterwards if need be.
func (r Responder) Download(fileName string, data io.Reader) {
	r.writer.Header().Set("Content-Type", fileNameToContentType(fileName))
	r.writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	r.writer.WriteHeader(http.StatusOK)

	if data == nil {
		return
	}

	_, err := io.Copy(r.writer, data)
	if err != nil {
		http.Error(r.writer, err.Error(), http.StatusInternalServerError)
	}
}

// Download delivers the file data to the client/caller in a way that indicates that it should
// be given a download prompt (if using a browser or some other UI-based client). The file name
// determines the Content-Type header we'll use in the response as well as be the default download
// name that the caller will be presented with in their client/browser.
func (r Responder) DownloadBytes(fileName string, data []byte) {
	r.Download(fileName, bytes.NewBuffer(data))
}

// Redirect performs a 307-style TEMPORARY redirect to the given resource. You can use printf-style
// formatting to make it easier to build the location you're redirecting to.
func (r Responder) Redirect(uri string, args ...interface{}) {
	http.Redirect(r.writer, r.request, fmt.Sprintf(uri, args...), http.StatusTemporaryRedirect)
}

// Redirect performs a 308-style PERMANENT redirect to the given resource. You can use printf-style
// formatting to make it easier to build the location you're redirecting to.
func (r Responder) RedirectPermanent(uri string, args ...interface{}) {
	http.Redirect(r.writer, r.request, fmt.Sprintf(uri, args...), http.StatusPermanentRedirect)
}

// NotModified writes a 304 response with no content. You typically will use this when performing
// ETag staleness checks and the like.
func (r Responder) NotModified(errs ...error) {
	if err := firstError(errs...); err != nil {
		r.Fail(err)
		return
	}
	r.writer.WriteHeader(http.StatusNotModified)
}

// Failure analyzes the failure for the endpoint and responds with the most appropriate
// 4XX/5XX status code and message for the error. It tries to unwrap the error looking for
// an error with either a Status(), StatusCode(), or Code() function (see the ErrorXXX
// interfaces in this package) to determine what HTTP status code we will try to fail with.
func (r Responder) Fail(err error) {
	errResponse := toErrorResponse(err)
	writeJSON(r.writer, errResponse.Status, errResponse)
}

// BadRequest responds w/ a 400 status and a body that contains the status/message.
func (r Responder) BadRequest(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusBadRequest, Message: msg})
}

// Unauthorized responds w/ a 401 status and a body that contains the status/message.
func (r Responder) Unauthorized(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusUnauthorized, Message: msg})
}

// Forbidden responds w/ a 403 status and a body that contains the status/message.
func (r Responder) Forbidden(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusForbidden, Message: msg})
}

// NotFound responds w/ a 404 status and a body that contains the status/message.
func (r Responder) NotFound(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusNotFound, Message: msg})
}

// MethodNotAllowed responds w/ a 405 status and a body that contains the status/message.
func (r Responder) MethodNotAllowed(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusMethodNotAllowed, Message: msg})
}

// Conflict responds w/ a 409 status and a body that contains the status/message.
func (r Responder) Conflict(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusConflict, Message: msg})
}

// TooManyRequests responds w/ a 429 status and a body that contains the status/message.
func (r Responder) TooManyRequests(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusTooManyRequests, Message: msg})
}

// InternalServerError responds w/ a 500 status and a body that contains the status/message.
func (r Responder) InternalServerError(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusInternalServerError, Message: msg})
}

// NotImplemented responds w/ a 501 status and a body that contains the status/message.
func (r Responder) NotImplemented(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusNotImplemented, Message: msg})
}

// BadGateway responds w/ a 502 status and a body that contains the status/message.
func (r Responder) BadGateway(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusBadGateway, Message: msg})
}

// ServiceUnavailable responds w/ a 503 status and a body that contains the status/message.
func (r Responder) ServiceUnavailable(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusServiceUnavailable, Message: msg})
}

// GatewayTimeout responds w/ a 504 status and a body that contains the status/message.
func (r Responder) GatewayTimeout(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusGatewayTimeout, Message: msg})
}

func writeJSON(res http.ResponseWriter, status int, value interface{}) {
	jsonBytes, _ := json.Marshal(value)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	_, _ = res.Write(jsonBytes)
}

// firstError grabs the first non-nil error in the given list of errors. This will return
// nil if there are no errors provided at all or if all of the errors are already nil.
func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func fileNameToContentType(fileName string) string {
	extPeriod := strings.LastIndex(fileName, ".")
	if extPeriod < 0 {
		return "application/octet-stream"
	}

	mimeType := mime.TypeByExtension(fileName[extPeriod:])
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}
