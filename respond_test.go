package respond_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/robsignorelli/respond"
	"github.com/stretchr/testify/suite"
)

func TestRespondSuite(t *testing.T) {
	suite.Run(t, new(RespondSuite))
}

type RespondSuite struct {
	suite.Suite
}

// TestNilInputs ensures that you supply a non-nil response writer. We don't fail if the
// request is nil for most replies since redirect is the only one that currently uses that
// value to handle its response.
func (suite RespondSuite) TestNilInputs() {
	// Everything needs a response writer.
	suite.Panics(func() {
		respond.To(nil, newRequest()).Ok("Hello")
	})
	suite.Panics(func() {
		respond.To(nil, nil).Ok("Hello")
	})
	suite.Panics(func() {
		respond.To(nil, nil).Created("Hello")
	})
	suite.Panics(func() {
		respond.To(nil, nil).Accepted("Hello")
	})
	suite.Panics(func() {
		respond.To(nil, nil).NoContent()
	})
	suite.Panics(func() {
		respond.To(nil, newRequest()).Redirect("https://google.com")
	})

	// Only redirect needs a request
	suite.NotPanics(func() {
		respond.To(newResponseWriter(), nil).Ok("Hello")
	})
	suite.Panics(func() {
		respond.To(newResponseWriter(), nil).Redirect("https://google.com")
	})
	suite.Panics(func() {
		respond.To(newResponseWriter(), nil).RedirectPermanent("https://google.com")
	})
}

func (suite RespondSuite) TestOk_nil() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Ok(nil)
	suite.assertStatus(w, 200)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertBody(w, "null") // the JSON value null, not the string null
}

func (suite RespondSuite) TestOk_string() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Ok("hello")
	suite.assertStatus(w, 200)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertBody(w, `"hello"`)
}

func (suite RespondSuite) TestOk_struct() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Ok(mockUser{Name: "Bob", ID: 42})
	suite.assertStatus(w, 200)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertJSON(w, "name", "Bob")
	suite.assertJSON(w, "id", 42)
}

func (suite RespondSuite) TestOk_pointer() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Ok(&mockUser{Name: "Bob", ID: 42})
	suite.assertStatus(w, 200)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertJSON(w, "name", "Bob")
	suite.assertJSON(w, "id", 42)
}

func (suite RespondSuite) TestCreated_string() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Created("hello")
	suite.assertStatus(w, 201)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertBody(w, `"hello"`)
}

func (suite RespondSuite) TestCreated_struct() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Created(mockUser{Name: "Bob", ID: 42})
	suite.assertStatus(w, 201)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertJSON(w, "name", "Bob")
	suite.assertJSON(w, "id", 42)
}

func (suite RespondSuite) TestCreated_pointer() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Created(&mockUser{Name: "Bob", ID: 42})
	suite.assertStatus(w, 201)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertJSON(w, "name", "Bob")
	suite.assertJSON(w, "id", 42)
}

func (suite RespondSuite) TestAccepted_nil() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Accepted(nil)
	suite.assertStatus(w, 202)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertBody(w, "null") // the JSON value null, not the string null
}

func (suite RespondSuite) TestAccepted_string() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Accepted("hello")
	suite.assertStatus(w, 202)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertBody(w, `"hello"`)
}

func (suite RespondSuite) TestAccepted_struct() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Accepted(mockUser{Name: "Bob", ID: 42})
	suite.assertStatus(w, 202)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertJSON(w, "name", "Bob")
	suite.assertJSON(w, "id", 42)
}

func (suite RespondSuite) TestAccepted_pointer() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Accepted(&mockUser{Name: "Bob", ID: 42})
	suite.assertStatus(w, 202)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertJSON(w, "name", "Bob")
	suite.assertJSON(w, "id", 42)
}

func (suite RespondSuite) TestNoContent() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).NoContent()
	suite.assertStatus(w, 204)
	suite.assertHeader(w, "Content-Type", "")
	suite.assertEmptyBody(w)
}

func (suite RespondSuite) TestNotModified() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).NotModified()
	suite.assertStatus(w, 304)
	suite.assertHeader(w, "Content-Type", "")
	suite.assertEmptyBody(w)
}

func (suite RespondSuite) TestRedirect_exact() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Redirect("https://google.com")
	suite.assertStatus(w, 307)
	suite.assertHeader(w, "Content-Type", "")
	suite.assertHeader(w, "Location", "https://google.com")
}

func (suite RespondSuite) TestRedirect_substitutions() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Redirect("https://google.com?q=%s", "hello")
	suite.assertStatus(w, 307)
	suite.assertHeader(w, "Content-Type", "")
	suite.assertHeader(w, "Location", "https://google.com?q=hello")
}

func (suite RespondSuite) TestRedirectPermanent_exact() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).RedirectPermanent("https://google.com")
	suite.assertStatus(w, 308)
	suite.assertHeader(w, "Content-Type", "")
	suite.assertHeader(w, "Location", "https://google.com")
}

func (suite RespondSuite) TestRedirectPermanent_substitutions() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).RedirectPermanent("https://google.com?q=%s", "hello")
	suite.assertStatus(w, 308)
	suite.assertHeader(w, "Content-Type", "")
	suite.assertHeader(w, "Location", "https://google.com?q=hello")
}

func (suite RespondSuite) TestFail_unknownCode() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Fail(nil)
	suite.assertStatus(w, 500)
	suite.assertJSON(w, "status", 500)
	suite.assertJSON(w, "message", "unknown error")

	w = newResponseWriter()
	req = newRequest()

	respond.To(w, req).Fail(fmt.Errorf("blah"))
	suite.assertStatus(w, 500)
	suite.assertJSON(w, "status", 500)
	suite.assertJSON(w, "message", "blah")
}

func (suite RespondSuite) TestBadRequest() {
	suite.runErrorTests(400, func(r respond.Responder) responderErrorFunc {
		return r.BadRequest
	})
}

func (suite RespondSuite) TestUnauthorized() {
	suite.runErrorTests(401, func(r respond.Responder) responderErrorFunc {
		return r.Unauthorized
	})
}

func (suite RespondSuite) TestForbidden() {
	suite.runErrorTests(403, func(r respond.Responder) responderErrorFunc {
		return r.Forbidden
	})
}

func (suite RespondSuite) TestNotFound() {
	suite.runErrorTests(404, func(r respond.Responder) responderErrorFunc {
		return r.NotFound
	})
}

func (suite RespondSuite) TestMethodNotAllowed() {
	suite.runErrorTests(405, func(r respond.Responder) responderErrorFunc {
		return r.MethodNotAllowed
	})
}

func (suite RespondSuite) TestConflict() {
	suite.runErrorTests(409, func(r respond.Responder) responderErrorFunc {
		return r.Conflict
	})
}

func (suite RespondSuite) TestTooManyRequests() {
	suite.runErrorTests(429, func(r respond.Responder) responderErrorFunc {
		return r.TooManyRequests
	})
}

func (suite RespondSuite) TestInternalServerError() {
	suite.runErrorTests(500, func(r respond.Responder) responderErrorFunc {
		return r.InternalServerError
	})
}

func (suite RespondSuite) TestNotImplemented() {
	suite.runErrorTests(501, func(r respond.Responder) responderErrorFunc {
		return r.NotImplemented
	})
}

func (suite RespondSuite) TestBadGateway() {
	suite.runErrorTests(502, func(r respond.Responder) responderErrorFunc {
		return r.BadGateway
	})
}

func (suite RespondSuite) TestServiceUnavailable() {
	suite.runErrorTests(503, func(r respond.Responder) responderErrorFunc {
		return r.ServiceUnavailable
	})
}

func (suite RespondSuite) TestGatewayTimeout() {
	suite.runErrorTests(504, func(r respond.Responder) responderErrorFunc {
		return r.GatewayTimeout
	})
}

func (suite RespondSuite) TestOptionalError_nil() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Ok("hello", nil)
	suite.assertStatus(w, 200)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertBody(w, `"hello"`)
}

func (suite RespondSuite) TestOptionalError_standard() {
	w := newResponseWriter()
	req := newRequest()
	respond.To(w, req).Ok("hello", fmt.Errorf("foo"))
	suite.assertError(w, 500, "foo")

	w = newResponseWriter()
	req = newRequest()
	respond.To(w, req).Created("hello", fmt.Errorf("bar"))
	suite.assertError(w, 500, "bar")

	w = newResponseWriter()
	req = newRequest()
	respond.To(w, req).Accepted("hello", fmt.Errorf("bar"))
	suite.assertError(w, 500, "bar")

	w = newResponseWriter()
	req = newRequest()
	respond.To(w, req).NoContent(fmt.Errorf("baz"))
	suite.assertError(w, 500, "baz")

	w = newResponseWriter()
	req = newRequest()
	respond.To(w, req).NotModified(fmt.Errorf("blah"))
	suite.assertError(w, 500, "blah")
}

func (suite RespondSuite) TestOptionalError_code() {
	w := newResponseWriter()
	req := newRequest()
	respond.To(w, req).Ok("hello", errorWithCode{status: 401, message: "moo"})
	suite.assertError(w, 401, "moo")

	w = newResponseWriter()
	req = newRequest()
	respond.To(w, req).Ok("hello", errorWithCode{status: 504, message: "boo"})
	suite.assertError(w, 504, "boo")
}

func (suite RespondSuite) TestOptionalError_status() {
	w := newResponseWriter()
	req := newRequest()
	respond.To(w, req).Ok("hello", errorWithStatus{status: 401, message: "moo"})
	suite.assertError(w, 401, "moo")

	w = newResponseWriter()
	req = newRequest()
	respond.To(w, req).Ok("hello", errorWithStatus{status: 504, message: "boo"})
	suite.assertError(w, 504, "boo")
}

func (suite RespondSuite) TestOptionalError_statusCode() {
	w := newResponseWriter()
	req := newRequest()
	respond.To(w, req).Ok("hello", errorWithStatusCode{status: 401, message: "moo"})
	suite.assertError(w, 401, "moo")

	w = newResponseWriter()
	req = newRequest()
	respond.To(w, req).Ok("hello", errorWithStatusCode{status: 504, message: "boo"})
	suite.assertError(w, 504, "boo")
}

func (suite RespondSuite) TestOptionalError_multiple() {
	w := newResponseWriter()
	req := newRequest()

	err1 := errorWithStatusCode{status: 401, message: "foo"}
	err2 := errorWithStatusCode{status: 402, message: "bar"}
	err3 := errorWithStatusCode{status: 403, message: "baz"}
	respond.To(w, req).Ok("hello", err1, err2, err3)
	suite.assertError(w, 401, "foo")
}

func (suite RespondSuite) TestServe_nil() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Serve("foo.txt", nil)

	suite.assertHeader(w, "Content-Type", "text/plain; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", "inline")
	suite.assertEmptyBody(w)
}

func (suite RespondSuite) TestServe_noExt() {
	w := newResponseWriter()
	req := newRequest()

	buf := &bytes.Buffer{}
	buf.WriteString("hello world")
	respond.To(w, req).Serve("foobarbaz", buf)

	suite.assertHeader(w, "Content-Type", "application/octet-stream")
	suite.assertHeader(w, "Content-Disposition", "inline")
	suite.assertBody(w, "hello world")
}

func (suite RespondSuite) TestServe_unknownExt() {
	w := newResponseWriter()
	req := newRequest()

	buf := &bytes.Buffer{}
	buf.WriteString("hello world")
	respond.To(w, req).Serve("foobarbaz.goblins", buf)

	suite.assertHeader(w, "Content-Type", "application/octet-stream")
	suite.assertHeader(w, "Content-Disposition", "inline")
	suite.assertBody(w, "hello world")
}

func (suite RespondSuite) TestServe_text() {
	w := newResponseWriter()
	req := newRequest()

	buf := &bytes.Buffer{}
	buf.WriteString("hello world")
	respond.To(w, req).Serve("foo.txt", buf)

	suite.assertHeader(w, "Content-Type", "text/plain; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", "inline")
	suite.assertBody(w, "hello world")
}

func (suite RespondSuite) TestServe_html() {
	w := newResponseWriter()
	req := newRequest()

	buf := &bytes.Buffer{}
	buf.WriteString("<p>hello world</p>")

	respond.To(w, req).Serve("foo.html", buf)

	suite.assertHeader(w, "Content-Type", "text/html; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", "inline")
	suite.assertBody(w, "<p>hello world</p>")
}

func (suite RespondSuite) TestServe_binary() {
	w := newResponseWriter()
	req := newRequest()

	buf := &bytes.Buffer{}
	buf.WriteByte(0x42)
	buf.WriteByte(0x43)
	buf.WriteByte(0x44)

	respond.To(w, req).Serve("foo.jpg", buf)

	suite.assertHeader(w, "Content-Type", "image/jpeg")
	suite.assertHeader(w, "Content-Disposition", "inline")
	suite.Equal([]byte{0x42, 0x43, 0x44}, w.Body)
}

func (suite RespondSuite) TestServe_error() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Serve("foo.txt", &bytes.Buffer{}, errorWithStatus{
		status:  504,
		message: "rats",
	})

	suite.assertError(w, 504, "rats")
}

func (suite RespondSuite) TestServe_readerFail() {
	w := newResponseWriter()
	req := newRequest()

	reader := badReader{failureStatus: 403}
	respond.To(w, req).Serve("foo.txt", reader)

	suite.assertError(w, 403, "bad monkey")
}

func (suite RespondSuite) TestServeBytes_nil() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).ServeBytes("foo.txt", nil)

	suite.assertHeader(w, "Content-Type", "text/plain; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", "inline")
	suite.assertEmptyBody(w)
}

func (suite RespondSuite) TestServeBytes_text() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).ServeBytes("foo.txt", []byte("hello world"))

	suite.assertHeader(w, "Content-Type", "text/plain; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", "inline")
	suite.assertBody(w, "hello world")
}

func (suite RespondSuite) TestServeBytes_html() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).ServeBytes("foo.html", []byte("<p>hello world</p>"))
	suite.assertHeader(w, "Content-Type", "text/html; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", "inline")
	suite.assertBody(w, "<p>hello world</p>")
}

func (suite RespondSuite) TestServeBytes_binary() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).ServeBytes("foo.jpg", []byte{0x42, 0x43, 0x44})

	suite.assertHeader(w, "Content-Type", "image/jpeg")
	suite.assertHeader(w, "Content-Disposition", "inline")
	suite.Equal([]byte{0x42, 0x43, 0x44}, w.Body)
}

func (suite RespondSuite) TestServeBytes_error() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).ServeBytes("foo.txt", []byte{}, errorWithStatus{
		status:  504,
		message: "rats",
	})

	suite.assertError(w, 504, "rats")
}

func (suite RespondSuite) TestDownload_nil() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Download("foo.txt", nil)

	suite.assertHeader(w, "Content-Type", "text/plain; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", `attachment; filename="foo.txt"`)
	suite.assertEmptyBody(w)
}

func (suite RespondSuite) TestDownload_text() {
	w := newResponseWriter()
	req := newRequest()

	buf := &bytes.Buffer{}
	buf.WriteString("hello world")
	respond.To(w, req).Download("foo.txt", buf)

	suite.assertHeader(w, "Content-Type", "text/plain; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", `attachment; filename="foo.txt"`)
	suite.assertBody(w, "hello world")
}

func (suite RespondSuite) TestDownload_html() {
	w := newResponseWriter()
	req := newRequest()

	buf := &bytes.Buffer{}
	buf.WriteString("<p>hello world</p>")

	respond.To(w, req).Download("foo.html", buf)

	suite.assertHeader(w, "Content-Type", "text/html; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", `attachment; filename="foo.html"`)
	suite.assertBody(w, "<p>hello world</p>")
}

func (suite RespondSuite) TestDownload_binary() {
	w := newResponseWriter()
	req := newRequest()

	buf := &bytes.Buffer{}
	buf.WriteByte(0x42)
	buf.WriteByte(0x43)
	buf.WriteByte(0x44)

	respond.To(w, req).Download("foo.jpg", buf)

	suite.assertHeader(w, "Content-Type", "image/jpeg")
	suite.assertHeader(w, "Content-Disposition", `attachment; filename="foo.jpg"`)
	suite.Equal([]byte{0x42, 0x43, 0x44}, w.Body)
}

func (suite RespondSuite) TestDownload_error() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Download("foo.txt", &bytes.Buffer{}, errorWithStatus{
		status:  504,
		message: "rats",
	})

	suite.assertError(w, 504, "rats")
}

func (suite RespondSuite) TestDownload_readerFail() {
	w := newResponseWriter()
	req := newRequest()

	reader := badReader{failureStatus: 403}
	respond.To(w, req).Download("foo.txt", reader)

	suite.assertError(w, 403, "bad monkey")
}

func (suite RespondSuite) TestDownloadBytes_nil() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).DownloadBytes("foo.txt", nil)

	suite.assertHeader(w, "Content-Type", "text/plain; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", `attachment; filename="foo.txt"`)
	suite.assertEmptyBody(w)
}

func (suite RespondSuite) TestDownloadBytes_text() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).DownloadBytes("foo.txt", []byte("hello world"))

	suite.assertHeader(w, "Content-Type", "text/plain; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", `attachment; filename="foo.txt"`)
	suite.assertBody(w, "hello world")
}

func (suite RespondSuite) TestDownloadBytes_html() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).DownloadBytes("foo.html", []byte("<p>hello world</p>"))
	suite.assertHeader(w, "Content-Type", "text/html; charset=utf-8")
	suite.assertHeader(w, "Content-Disposition", `attachment; filename="foo.html"`)
	suite.assertBody(w, "<p>hello world</p>")
}

func (suite RespondSuite) TestDownloadBytes_binary() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).DownloadBytes("foo.jpg", []byte{0x42, 0x43, 0x44})

	suite.assertHeader(w, "Content-Type", "image/jpeg")
	suite.assertHeader(w, "Content-Disposition", `attachment; filename="foo.jpg"`)
	suite.Equal([]byte{0x42, 0x43, 0x44}, w.Body)
}

func (suite RespondSuite) TestDownloadBytes_error() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).DownloadBytes("foo.txt", []byte{}, errorWithStatus{
		status:  504,
		message: "rats",
	})

	suite.assertError(w, 504, "rats")
}

/*
 * -------- Assertion Helpers ------------------
 */

type responderErrorFunc func(string, ...interface{})
type responderErrorFactory func(responder respond.Responder) responderErrorFunc

func (suite RespondSuite) runErrorTests(expectedCode int, errorResponse responderErrorFactory) {
	w := newResponseWriter()
	req := newRequest()
	errorResponse(respond.To(w, req))("")
	suite.assertError(w, expectedCode, "")

	w = newResponseWriter()
	req = newRequest()
	errorResponse(respond.To(w, req))("foo")
	suite.assertError(w, expectedCode, "foo")

	w = newResponseWriter()
	req = newRequest()
	errorResponse(respond.To(w, req))("foo %s %v", "bar", 42)
	suite.assertError(w, expectedCode, "foo bar 42")
}

func (suite RespondSuite) assertStatus(res *mockResponseWriter, expected int) {
	suite.Require().Equal(expected, res.StatusCode)
}

func (suite RespondSuite) assertHeader(res *mockResponseWriter, headerName, expectedValue string) {
	suite.Require().Equal(expectedValue, res.Header().Get(headerName))
}

func (suite RespondSuite) assertBody(res *mockResponseWriter, expected string) {
	suite.Require().Equal([]byte(expected), res.Body)
}

func (suite RespondSuite) assertEmptyBody(res *mockResponseWriter) {
	suite.Require().Len(res.Body, 0)
}

func (suite RespondSuite) assertError(w *mockResponseWriter, status int, message string) {
	suite.assertStatus(w, status)
	suite.assertHeader(w, "Content-Type", "application/json")
	suite.assertJSON(w, "status", status)
	if message == "" {
		suite.Require().NotContains(string(w.Body), `\"message\"`)
	} else {
		suite.assertJSON(w, "message", message)
	}
}

func (suite RespondSuite) assertJSON(res *mockResponseWriter, field string, value interface{}) {
	switch value.(type) {
	case string:
		jsonText := fmt.Sprintf(`"%s":"%v"`, field, value)
		suite.Require().Contains(string(res.Body), jsonText)
	default:
		jsonText := fmt.Sprintf(`"%s":%v`, field, value)
		suite.Require().Contains(string(res.Body), jsonText)
	}
}

/*
 * -------- Custom Response/Error Types ------------------
 */

type mockResponseWriter struct {
	Headers    http.Header
	StatusCode int
	Body       []byte
}

func (w *mockResponseWriter) Header() http.Header {
	return w.Headers
}

func (w *mockResponseWriter) Write(data []byte) (int, error) {
	w.Body = append(w.Body, data...)
	return len(data), nil
}

func (w *mockResponseWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
}

func newResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		Headers: map[string][]string{},
	}
}

func newRequest() *http.Request {
	return &http.Request{}
}

type mockUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type errorWithStatusCode struct {
	status  int
	message string
}

func (err errorWithStatusCode) Error() string {
	return err.message
}

func (err errorWithStatusCode) StatusCode() int {
	return err.status
}

type errorWithCode struct {
	status  int
	message string
}

func (err errorWithCode) Error() string {
	return err.message
}

func (err errorWithCode) Code() int {
	return err.status
}

type errorWithStatus struct {
	status  int
	message string
}

func (err errorWithStatus) Error() string {
	return err.message
}

func (err errorWithStatus) Status() int {
	return err.status
}

type badReader struct {
	failureStatus int
}

func (b badReader) Read(_ []byte) (n int, err error) {
	return 0, errorWithStatus{
		status:  b.failureStatus,
		message: "bad monkey",
	}
}
