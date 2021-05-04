package respond_test

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"testing"

	"github.com/monadicstack/respond"
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

func (suite RespondSuite) TestHTML() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).HTML("<p>hello <b>world</b></p>")
	suite.assertStatus(w, 200)
	suite.assertHeader(w, "Content-Type", "text/html; charset=utf-8")
	suite.assertBody(w, "<p>hello <b>world</b></p>")
}

func (suite RespondSuite) TestHTML_empty() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).HTML("")
	suite.assertStatus(w, 200)
	suite.assertHeader(w, "Content-Type", "text/html; charset=utf-8")
	suite.assertEmptyBody(w)
}

func (suite RespondSuite) TestHTML_error() {
	w := newResponseWriter()
	req := newRequest()

	err := errorWithStatus{
		status:  503,
		message: "it's dead jim",
	}
	respond.To(w, req).HTML("<p>hello <b>world</b></p>", err)
	suite.assertError(w, 503, "it's dead jim")
}

func (suite RespondSuite) TestHTMLTemplate() {
	w := newResponseWriter()
	req := newRequest()

	temp := template.Must(template.New("HTMLTemplate").Parse(`<p>{{ . }} is {{ . }}</p>`))
	respond.To(w, req).HTMLTemplate(temp, "Bob")

	suite.assertBody(w, "<p>Bob is Bob</p>")
}

func (suite RespondSuite) TestHTMLTemplate_error() {
	w := newResponseWriter()
	req := newRequest()

	temp := template.Must(template.New("HTMLTemplate").Parse(`<p>{{ . }} is {{ . }}</p>`))
	err := errorWithStatus{
		status:  503,
		message: "it's dead jim",
	}
	respond.To(w, req).HTMLTemplate(temp, "Bob", err)

	suite.assertError(w, 503, "it's dead jim")
}

func (suite RespondSuite) TestHTMLTemplate_nilTemplate() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).HTMLTemplate(nil, "Bob")
	suite.assertStatus(w, 200)
	suite.assertEmptyBody(w)
}

// When the template fails to evaluate, you should get a 500 error.
func (suite RespondSuite) TestHTMLTemplate_evalError() {
	w := newResponseWriter()
	req := newRequest()

	temp := template.Must(template.New("HTMLTemplate").Parse(`<p>{{ .Foo }} is {{ . }}</p>`))
	respond.To(w, req).HTMLTemplate(temp, "Bob")

	suite.assertStatus(w, 500)
}

func (suite RespondSuite) TestRedirect_empty() {
	w := newResponseWriter()
	req := newRequest()

	// We don't really care about what the error "says" as long as it fails w/ a 500
	respond.To(w, req).Redirect("")
	suite.assertStatus(w, 500)
}

func (suite RespondSuite) TestRedirect_emptyAfterSubstitution() {
	w := newResponseWriter()
	req := newRequest()

	// We don't really care about what the error "says" as long as it fails w/ a 500
	respond.To(w, req).Redirect("%s%s", "", "")
	suite.assertStatus(w, 500)
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

func (suite RespondSuite) TestRedirectTo_nil() {
	w := newResponseWriter()
	req := newRequest()

	// We don't really care about what the error "says" as long as it fails w/ a 500
	respond.To(w, req).RedirectTo(nil)
	suite.assertStatus(w, 500)
}

func (suite RespondSuite) TestRedirectTo_valid() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).RedirectTo(fakeRedirector{URL: "https://google.com/foo"})
	suite.assertStatus(w, 307)
	suite.assertHeader(w, "Location", "https://google.com/foo")
}

func (suite RespondSuite) TestRedirectTo_error() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).RedirectTo(fakeRedirector{URL: "https://google.com/foo"}, errorWithStatus{
		status:  403,
		message: "nope",
	})
	suite.assertError(w, 403, "nope")
}

// Should ignore the fact that the Redirector is nil when there's an error present.
func (suite RespondSuite) TestRedirectTo_errorNilInput() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).RedirectTo(nil, errorWithStatus{
		status:  403,
		message: "nope",
	})
	suite.assertError(w, 403, "nope")
}

func (suite RespondSuite) TestRedirectPermanent_empty() {
	w := newResponseWriter()
	req := newRequest()

	// We don't really care about what the error "says" as long as it fails w/ a 500
	respond.To(w, req).RedirectPermanent("")
	suite.assertStatus(w, 500)
}

func (suite RespondSuite) TestRedirectPermanent_emptyAfterSubstitution() {
	w := newResponseWriter()
	req := newRequest()

	// We don't really care about what the error "says" as long as it fails w/ a 500
	respond.To(w, req).RedirectPermanent("%s%s", "", "")
	suite.assertStatus(w, 500)
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

func (suite RespondSuite) TestRedirectPermanentTo_nil() {
	w := newResponseWriter()
	req := newRequest()

	// We don't really care about what the error "says" as long as it fails w/ a 500
	respond.To(w, req).RedirectPermanentTo(nil)
	suite.assertStatus(w, 500)
}

func (suite RespondSuite) TestRedirectPermanentTo_valid() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).RedirectPermanentTo(fakeRedirector{URL: "https://google.com/foo"})
	suite.assertStatus(w, 308)
	suite.assertHeader(w, "Location", "https://google.com/foo")
}

func (suite RespondSuite) TestRedirectPermanentTo_error() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).RedirectPermanentTo(fakeRedirector{URL: "https://google.com/foo"}, errorWithStatus{
		status:  403,
		message: "nope",
	})
	suite.assertError(w, 403, "nope")
}

// Should ignore the fact that the Redirector is nil when there's an error present.
func (suite RespondSuite) TestRedirectPermanentTo_errorNilInput() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).RedirectPermanentTo(nil, errorWithStatus{
		status:  403,
		message: "nope",
	})
	suite.assertError(w, 403, "nope")
}

// Should allow you to write "raw" results by implementing io.Reader.
func (suite RespondSuite) TestRaw_empty() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Ok(bytes.NewBufferString(""))
	suite.assertStatus(w, 200)
	suite.assertRaw(w, "")
	suite.assertHeader(w, "Content-Type", "application/octet-stream")
	suite.assertHeader(w, "Content-Disposition", "inline")
}

// Should allow you to write "raw" results to non-200 status success codes.
func (suite RespondSuite) TestRaw_status() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Created(bytes.NewBufferString(""))
	suite.assertStatus(w, 201)
	suite.assertRaw(w, "")
	suite.assertHeader(w, "Content-Type", "application/octet-stream")
	suite.assertHeader(w, "Content-Disposition", "inline")
}

// Should allow you to write "raw" string results by using an io.Reader.
func (suite RespondSuite) TestRaw_text() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Ok(bytes.NewBufferString("hello world"))
	suite.assertStatus(w, 200)
	suite.assertRaw(w, "hello world")
	suite.assertHeader(w, "Content-Type", "application/octet-stream")
	suite.assertHeader(w, "Content-Disposition", "inline")
}

// Should allow you to write "raw" binary results by using an io.Reader.
func (suite RespondSuite) TestRaw_binary() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Ok(bytes.NewBuffer([]byte{0x42, 0x43, 0x44}))
	suite.assertStatus(w, 200)
	suite.assertRaw(w, string([]byte{0x42, 0x43, 0x44}))
	suite.assertHeader(w, "Content-Type", "application/octet-stream")
	suite.assertHeader(w, "Content-Disposition", "inline")
}

// Should allow you to specify the Content-Type by implementing ContentTypeSpecifier.
func (suite RespondSuite) TestRaw_contentType() {
	type R struct {
		rawReader
		rawContentType
	}

	w := newResponseWriter()
	req := newRequest()

	result := R{}
	result.reader = bytes.NewBufferString("Do you see what happens, Larry?")
	result.contentType = "text/plain"

	respond.To(w, req).Ok(result)
	suite.assertStatus(w, 200)
	suite.assertRaw(w, "Do you see what happens, Larry?")
	suite.assertHeader(w, "Content-Type", "text/plain")
	suite.assertHeader(w, "Content-Disposition", "inline")
}

// When implementing ContentTypeSpecifier, should use default type when returning "".
func (suite RespondSuite) TestRaw_contentTypeEmpty() {
	type R struct {
		rawReader
		rawContentType
	}

	w := newResponseWriter()
	req := newRequest()

	result := R{}
	result.reader = bytes.NewBufferString("Do you see what happens, Larry?")

	respond.To(w, req).Ok(result)
	suite.assertStatus(w, 200)
	suite.assertRaw(w, "Do you see what happens, Larry?")
	suite.assertHeader(w, "Content-Type", "application/octet-stream")
	suite.assertHeader(w, "Content-Disposition", "inline")
}

// Should allow you to specify the Content-Disposition by implementing FileNameSpecifier.
func (suite RespondSuite) TestRaw_contentDisposition() {
	type R struct {
		rawReader
		rawFileName
	}

	w := newResponseWriter()
	req := newRequest()

	result := R{}
	result.reader = bytes.NewBufferString("Do you see what happens, Larry?")
	result.fileName = "stranger.log"

	respond.To(w, req).Ok(result)
	suite.assertStatus(w, 200)
	suite.assertRaw(w, "Do you see what happens, Larry?")
	suite.assertHeader(w, "Content-Type", "application/octet-stream")
	suite.assertHeader(w, "Content-Disposition", `attachment; filename="stranger.log"`)
}

// When implementing FileNameSpecifier, should still be "inline" if you return "".
func (suite RespondSuite) TestRaw_contentDispositionEmpty() {
	type R struct {
		rawReader
		rawFileName
	}

	w := newResponseWriter()
	req := newRequest()

	result := R{}
	result.reader = bytes.NewBufferString("Do you see what happens, Larry?")

	respond.To(w, req).Ok(result)
	suite.assertStatus(w, 200)
	suite.assertRaw(w, "Do you see what happens, Larry?")
	suite.assertHeader(w, "Content-Type", "application/octet-stream")
	suite.assertHeader(w, "Content-Disposition", "inline")
}

// Should allow you to specify the Content-Type AND Content-Disposition by implementing
// all of the necessary types.
func (suite RespondSuite) TestRaw_contentTypeAndDisposition() {
	type R struct {
		rawReader
		rawFileName
		rawContentType
	}

	w := newResponseWriter()
	req := newRequest()

	result := R{}
	result.reader = bytes.NewBufferString("Do you see what happens, Larry?")
	result.contentType = "text/plain"
	result.fileName = "stranger.log"

	respond.To(w, req).Ok(result)
	suite.assertStatus(w, 200)
	suite.assertRaw(w, "Do you see what happens, Larry?")
	suite.assertHeader(w, "Content-Type", "text/plain")
	suite.assertHeader(w, "Content-Disposition", `attachment; filename="stranger.log"`)
}

// If your raw reader is also an io.Closer, make sure to close it after responding automiatically.
func (suite RespondSuite) TestRaw_close() {
	type R struct {
		rawReader
		rawCloser
	}

	w := newResponseWriter()
	req := newRequest()

	result := R{}
	result.reader = bytes.NewBufferString("Do you see what happens, Larry?")

	respond.To(w, req).Ok(&result)
	suite.Require().True(result.closed, "Raw result was not automatically closed despite implementing io.Closer")
}

func (suite RespondSuite) TestJSON_unableToMarshal() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Ok(make(chan int, 5))
	suite.assertStatus(w, 500)
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

func (suite RespondSuite) TestGone() {
	suite.runErrorTests(410, func(r respond.Responder) responderErrorFunc {
		return r.Gone
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

func (suite RespondSuite) TestReply_redirect() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Reply(200, fakeRedirector{URL: "https://google.com"})
	suite.assertStatus(w, 307)
	suite.assertHeader(w, "Location", "https://google.com")
	suite.assertEmptyBody(w)
}

func (suite RespondSuite) TestReply_redirect_forceStatus() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Reply(308, fakeRedirector{URL: "https://google.com"})
	suite.assertStatus(w, 307)
	suite.assertHeader(w, "Location", "https://google.com")
	suite.assertEmptyBody(w)
}

func (suite RespondSuite) TestReply_redirect_error() {
	w := newResponseWriter()
	req := newRequest()

	respond.To(w, req).Reply(308, fakeRedirector{URL: "https://google.com"}, fmt.Errorf("crap"))
	suite.assertError(w, 500, "crap")
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

func (suite RespondSuite) assertRaw(res *mockResponseWriter, rawData string) {
	suite.Require().Equal(string(res.Body), rawData)
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

type fakeRedirector struct {
	URL string
}

func (r fakeRedirector) Redirect() string {
	return r.URL
}

type rawReader struct {
	reader io.Reader
}

func (r rawReader) Read(buf []byte) (int, error) {
	return r.reader.Read(buf)
}

type rawCloser struct {
	closed bool
}

func (r *rawCloser) Close() error {
	r.closed = true
	return nil
}

type rawContentType struct {
	contentType string
}

func (r rawContentType) ContentType() string {
	return r.contentType
}

type rawFileName struct {
	fileName string
}

func (r rawFileName) FileName() string {
	return r.fileName
}
