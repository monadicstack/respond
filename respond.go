package respond

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"strings"
)

// To creates a "Responder" that replies to the inputs for the given HTTP request. For style/consistency
// purposes, this should be the first line of your HTTP handler: `response := responder.To(w, req)`
func To(w http.ResponseWriter, req *http.Request) Responder {
	return Responder{writer: w, request: req}
}

// Redirector defines a type that your handler can "return" to one of the responder functions to indicate that this
// should be a redirect response instead of the standard 2XX style response you intended. This can also be used in
// general purpose Reply() calls to trigger redirects as well.
type Redirector interface {
	// Redirect returns the URL that you want the response to redirect to.
	Redirect() string
}

// ContentTypeSpecified provides details about a file-based response to indicate what we should
// use as the "Content-Type" header. Any io.Reader that 'respond' comes across will be
// treated as raw bytes, not a JSON-marshaled payload. By default, the Content-Type of the response
// will be "application/octet-stream", but if your result implements this interface, you can tell the
// responder what type to use instead. For instance, if the result is a JPG, you can have your result
// return "image/jpeg" and 'respond' will use that in the header instead of octet-stream.
type ContentTypeSpecified interface {
	// ContentType returns the "Content-Type" header you want to apply to the HTTP response. This
	// only applies when the result is an io.Reader, so you're returning raw results.
	ContentType() string
}

// FileNameSpecified provides the 'filename' details to use when filling out the HTTP Content-Disposition
// header. Any io.Reader that 'respond' comes across will be treated as raw bytes, not a JSON-marshaled
// payload. By default, 'respond' will specify "inline" for all raw responses (great for images and
// scripts you want to display inline in your UI).
//
// If you implement this interface, you can change the behavior to have the browser/client trigger a
// download of this asset instead. The file name you return here will dictate the default file name
// proposed by the save dialog.
type FileNameSpecified interface {
	// FileName triggers an attachment-style value for the Content-Disposition header when writing
	// raw HTTP responses. When this returns an empty string, the response's disposition should
	// be "inline". When it's any other value, it will be "attachment; filename=" with this value.
	//
	// This only applies when the result is an io.Reader, so you're returning raw results.
	FileName() string
}

// Responder provides helper functions for marshaling Go values/streams to send back to the user as well as
// applying the correct status code and headers. It's the core data structure for this package.
type Responder struct {
	writer  http.ResponseWriter
	request *http.Request
}

// Reply lets you respond with the custom status code of your choice and a JSON-marshaled version of your value.
func (r Responder) Reply(status int, value interface{}, errs ...error) {
	// Assume that any error we receive indicates that the operation failed, so respond accordingly.
	if err := firstError(errs...); err != nil {
		r.Fail(err)
		return
	}

	switch v := value.(type) {
	case Redirector:
		// The value you're returning is telling us redirect to another URL instead.
		r.Redirect(v.Redirect())
	case io.Reader:
		// The value looks like a file or some other raw, non-JSON content
		writeRaw(r.writer, status, v)
	default:
		// It's just some returned value that we should marshal as JSON and send back.
		writeJSON(r.writer, status, value)
	}
}

// Ok writes a 200 style response to the caller by marshalling the given raw value. If
// you provided an error, we'll ignore the value and return the appropriate 4XX/5XX
// response instead.
func (r Responder) Ok(value interface{}, errs ...error) {
	r.Reply(http.StatusOK, value, errs...)
}

// Created writes a 201 style response to the caller by marshalling the given raw value. If
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

// HTML returns 200 status code with the given "text/html" response body. If you provided an error,
// we'll ignore the value and return the appropriate 4XX/5XX response instead.
func (r Responder) HTML(markup string, errs ...error) {
	if err := firstError(errs...); err != nil {
		r.Fail(err)
		return
	}

	r.writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	r.writer.WriteHeader(http.StatusOK)
	_, _ = r.writer.Write([]byte(markup))
}

// HTMLTemplate accepts your pre-parsed html template and evaluates it using the given context value. All of
// the bytes generated by the template will be written directly to the response writer. If you provided an error,
// we'll return the appropriate 4XX/5XX response instead.
func (r Responder) HTMLTemplate(htmlTemplate *template.Template, ctxValue interface{}, errs ...error) {
	if err := firstError(errs...); err != nil {
		r.Fail(err)
		return
	}

	r.writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	r.writer.WriteHeader(http.StatusOK)

	if htmlTemplate == nil {
		return
	}

	err := htmlTemplate.Execute(r.writer, ctxValue)
	if err != nil {
		r.Fail(err)
	}
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
func (r Responder) Serve(fileName string, data io.Reader, errs ...error) {
	if err := firstError(errs...); err != nil {
		r.Fail(err)
		return
	}

	r.writer.Header().Set("Content-Type", fileNameToContentType(fileName))
	r.writer.Header().Set("Content-Disposition", "inline")
	r.writer.WriteHeader(http.StatusOK)

	if data == nil {
		return
	}

	_, err := io.Copy(r.writer, data)
	if err != nil {
		r.Fail(err)
	}
}

// ServeBytes responds with some sort of file data in an inline fashion. This lets you deliver
// things like inline images or videos or any other content that you want your callers/clients
// to embed directly in the client. The file name in this case case is simply used to determine
// the proper Content-Type to include in the response.
func (r Responder) ServeBytes(fileName string, data []byte, errs ...error) {
	r.Serve(fileName, bytes.NewBuffer(data), errs...)
}

// Download delivers the file data to the client/caller in a way that indicates that it should
// be given a download prompt (if using a browser or some other UI-based client). The file name
// determines the Content-Type header we'll use in the response as well as be the default download
// name that the caller will be presented with in their client/browser.
//
// It will read your 'data' stream to completion but it will still be up to you to Close() it
// afterwards if need be.
func (r Responder) Download(fileName string, data io.Reader, errs ...error) {
	if err := firstError(errs...); err != nil {
		r.Fail(err)
		return
	}

	r.writer.Header().Set("Content-Type", fileNameToContentType(fileName))
	r.writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	r.writer.WriteHeader(http.StatusOK)

	if data == nil {
		return
	}

	_, err := io.Copy(r.writer, data)
	if err != nil {
		r.Fail(err)
	}
}

// DownloadBytes delivers the file data to the client/caller in a way that indicates that it should
// be given a download prompt (if using a browser or some other UI-based client). The file name
// determines the Content-Type header we'll use in the response as well as be the default download
// name that the caller will be presented with in their client/browser.
func (r Responder) DownloadBytes(fileName string, data []byte, errs ...error) {
	r.Download(fileName, bytes.NewBuffer(data), errs...)
}

// Redirect performs a 307-style TEMPORARY redirect to the given resource. You can use printf-style
// formatting to make it easier to build the location you're redirecting to.
func (r Responder) Redirect(uriFormat string, args ...interface{}) {
	uri := fmt.Sprintf(uriFormat, args...)
	if uri == "" {
		r.Fail(fmt.Errorf("unable to redirect to empty url"))
		return
	}
	http.Redirect(r.writer, r.request, uri, http.StatusTemporaryRedirect)
}

// RedirectTo performs a 307-style TEMPORARY redirect to the URL returned by calling Redirect() on your value.
func (r Responder) RedirectTo(redirector Redirector, errs ...error) {
	if err := firstError(errs...); err != nil {
		r.Fail(err)
		return
	}
	if redirector == nil {
		r.Fail(fmt.Errorf("unable to redirect using nil redirector"))
		return
	}
	r.Redirect(redirector.Redirect())
}

// RedirectPermanent performs a 308-style PERMANENT redirect to the given resource. You can use printf-style
// formatting to make it easier to build the location you're redirecting to.
func (r Responder) RedirectPermanent(uriFormat string, args ...interface{}) {
	uri := fmt.Sprintf(uriFormat, args...)
	if uri == "" {
		r.Fail(fmt.Errorf("unable to redirect to empty url"))
		return
	}
	http.Redirect(r.writer, r.request, uri, http.StatusPermanentRedirect)
}

// RedirectPermanentTo performs a 308-style PERMANENT redirect to the URL returned by calling Redirect() on your value.
func (r Responder) RedirectPermanentTo(redirector Redirector, errs ...error) {
	if err := firstError(errs...); err != nil {
		r.Fail(err)
		return
	}
	if redirector == nil {
		r.Fail(fmt.Errorf("unable to redirect using nil redirector"))
		return
	}
	r.RedirectPermanent(redirector.Redirect())
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

// Fail accepts the error generated by your handler and responds with the most appropriate
// 4XX/5XX status code and message for that error. It tries to unwrap the error looking for
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

// Gone responds w/ a 410 status and a body that contains the status/message.
func (r Responder) Gone(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	r.Fail(errorResponse{Status: http.StatusGone, Message: msg})
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

// writeJSON marshals the result 'value' as JSON and writes the bytes to the response.
func writeJSON(res http.ResponseWriter, status int, value interface{}) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		http.Error(res, "json marshal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	_, _ = res.Write(jsonBytes)
}

// writeRaw accepts a reader containing the bytes of some file or raw set of data that the
// user wants to write to the caller.
func writeRaw(res http.ResponseWriter, status int, value io.Reader) {
	if closer, ok := value.(io.Closer); ok {
		defer func() { _ = closer.Close() }()
	}

	res.Header().Set("Content-Type", rawContentType(value))
	res.Header().Set("Content-Disposition", rawContentDisposition(value))
	res.WriteHeader(status)
	_, _ = io.Copy(res, value)
}

// rawContentType assumes "application/octet-stream" unless the return value implements
// the ContentTypeSpecified interface. In that case, this will return the content type
// that the reader specifies. The result is a valid value for the HTTP "Content-Type" header.
func rawContentType(value io.Reader) string {
	contentTyped, ok := value.(ContentTypeSpecified)
	if !ok {
		return "application/octet-stream"
	}

	contentType := contentTyped.ContentType()
	if contentType == "" {
		return "application/octet-stream"
	}

	return contentType
}

// rawContentDisposition returns an appropriate value for the "Content-Disposition"
// HTTP header. In most cases, this will return "inline", but if the reader implements
// the FileNameSpecified interface, this will return "attachment; filename=" with the
// reader's name specified.
func rawContentDisposition(value io.Reader) string {
	named, ok := value.(FileNameSpecified)
	if !ok {
		return "inline"
	}

	fileName := named.FileName()
	if fileName == "" {
		return "inline"
	}

	return `attachment; filename="` + fileName + `"`
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

// fileNameToContentType to take a file name/path and analyzes the file extension. With that extension, this
// will return the most relevant mime encoding type string. For example "foo/bar/baz.jpg" will return "image/jpeg".
// This returns "application/octet-stream" for any file name that doesn't have an extension or for any
// extension that the OS doesn't have a mime mapping for.
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
