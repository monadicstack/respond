# Respond

[![Go Report Card](https://goreportcard.com/badge/github.com/monadicstack/respond)](https://goreportcard.com/report/github.com/monadicstack/respond)

This package reduces the verbosity associated with responding to HTTP
requests when using standard `net/http` handlers (with a bent towards JSON
based REST APIs). Most HTTP handling
examples using the standard library tend to be 25% handling logic and
75% dealing with data marshaling, header management, status code updates,
etc. Respond tries to flip that ratio so more of your code is business
logic while still giving you robust, readable, maintainable response handling.

You focus on writing awesome code that does something meaningful
and `respond` takes care of the messy code that figures out how to
send your result to the caller.

### Getting Started

```
go get -u github.com/monadicstack/respond
```

### Basic Example

Here is a sample HTTP handler using the standard library.

```go
func MyHandler(w http.ResponseWriter, req *http.Request) {
    userID := param(req, "user")
    user, err := userRepo.FindById(userID)

    if err != nil {
	    http.Error(w, err.Error(), 500)
	    return
    }
    jsonBytes, err := json.Marshal(user)
    if err != nil {
    	http.Error(w, err.Error(), 500)
    	return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200)
    w.Write(jsonBytes)
}
```

Here is the same handler using the `respond` package. Create a
`Responder` for your writer/request pair and use the aptly-named
functions that line up to the type of HTTP response you want.

```go
func MyHandler(w http.ResponseWriter, req *http.Request) {
    response := respond.To(w, req)

    userID := param(req, "user")
    user, err := userRepo.FindById(userID)

    // If you want a 200 status...
    response.Ok(user, err)
}
```

### Success Responses

Responders have named helpers for some of the more common success
codes you might want to respond with:

```go
response := respond.To(w, req)
...

// Responds w/ a 200 and 'someValue' as JSON 
response.Ok(someValue)

// Responds w/ a 201 and 'someValue' as JSON 
response.Created(someValue)

// Responds w/ a 202 and 'someValue' as JSON 
response.Accept(someValue)

// Responds w/ a 204 and no body. 
response.NoContent()

// Responds w/ a 304 and no body. 
response.NotModified()
```

### Error Handling

The `Responder` type has a bunch of helpful functions for responding
with meaningful error statuses and messages. All error messages
support `printf` style formatting.

```go
response := respond.To(w, req)
...
// Status => 401
// Body   => { "status": 401, "message": "what, no credentials?" }
response.Unauthorized("what, no credentials?")

// Status => 403
// Body   => { "status": 403, "message": "missing 'badass' role" }
response.Forbidden("missing '%s' role", someRole)

// Status => 404
// Body   => { "status": 404, "message": "unable to find user [123]" }
response.NotFound("unable to find user [%s]", userID)

// Status => 500
// Body   => { "status": 500, "message": "what did you do!?!" }
response.InternalServerError("what did you do!?!")

// Status => 503
// Body   => { "status": 503, "message": "not working right now" }
response.ServiceUnavailable("not working right now")

```
These are just a few common ones. Check the docs or use auto-complete
in your IDE to see which errors are supported.

Here's a common pattern for utilizing these response functions if
you use named errors. It gives you strongly coded errors while
left-aligning your code to help keep your handler code more
idiomatic. If you have more control over the errors you generate,
take a look at the next section to see how you can reduce this
boilerplate even further.

```go
func MyHandler(w http.ResponseWriter, req *http.Request) {
    response := respond.To(w, req)

    userID := param(req, "user")
    user, err := userRepo.FindById(userID)

    switch err {
    case ErrNoSuchAccount:
        response.NotFound("no such users: %s", userID)
    case ErrDatabaseConn:
        response.ServiceUnavailable("user database unavailable")
    default:
        response.Ok(user, err)
    }
}
```

### Error Handling: Shorthand

While it has its places, writing those `switch` statements to respond
with the correct status code can be a pain. All of the success
response functions like `Ok()`, `Accepted()`, etc accept an optional
`Error` argument (yes... by bastardizing variadic functions... sorry).

```go
func MyHandler(w http.ResponseWriter, req *http.Request) {
    response := respond.To(w, req)

    userID := param(req, "user")
    user, err := userRepo.FindById(userID)

    // When 'err' is nil, the result will be a 200 w/ the user
    // data as JSON. When 'err' is non-nil, we'll ignore the
    // user data and return a 4XX/5XX error w/ err's message.
    response.Ok(user, err)
}
```

#### How Does It Know Which 4XX/5XX Status To Use?

`respond` uses Go 1.13 error unwrapping to detect if your error
conforms to one of these three error interfaces:

* `ErrorWithCode`
* `ErrorWithStatus`
* `ErrorWithStatusCode`

These are all `Error` interfaces that also contain either of
these functions:

* `func Code() int`
* `func Status() int`
* `func StatusCode() int`
  
This allows you to use your own custom error types, and as long
as they have one of these 3 functions, `respond` will use that
status code in the error.

Here is a sample error you might write that satisfies the
`ErrorWithCode` interface.

```go
type NotFoundError struct {
    message string
}
func (err NotFoundError) Error() string {
    return err.message
}
func (err NotFoundError) Code() int {
    return 404
}
```

Your business logic can just naturally return meaningful errors.

```go
func (repo *UserRepo) FindById(userID string) (*User, error) {
    user := User{}
    row := repo.db.Query("...")
    err := row.Scan(...)

    switch err {
    case sql.ErrNoRows:
        return nil, NotFoundError{message: "no such user"} 
    default:
        return user, err
}
```

Now your handler will return a 404 when the user doesn't exist and
fall back to a good 'ol 500 on any other type of error. Like before,
if there was no error, this will result in a 200 w/ the user data.

```go
func MyHandler(w http.ResponseWriter, req *http.Request) {
    response := respond.To(w, req)

    userID := param(req, "user")
    user, err := userRepo.FindById(userID)

    response.Ok(user, err)
}
```

### Redirects

Depending on what will make your handler more clear, you have two
options for triggering an HTTP redirect response:

```go
func MyHandler(w http.ResponseWriter, req *http.Request) {
    response := respond.To(w, req)

    fileID := param(req, "file")
    file, ok := fileStore.GetFileInfo(fileID)
    if !ok {
        response.NotFound("file not found: %s", fileID)
        return
    }

    response.Redirect("https://%s.s3.amazonaws.com/%s/%s",
    	file.Bucket,
    	file.Directory,
    	file.Name,
    )
}
```

This is fine when your redirects are simple, but the more complex
your substitutions are, the more you lose clarity because your focus
is drawn to the URL substitution rather than the actual business logic.

Alternately, you can use `RedirectTo()` and pass any value that implements
the `Redirector` interface; basically can return the fully-formed URL
that you want to redirect to. This can help clean up your handlers by
moving the URL building logic elsewhere, so your handler stays lean
and mean:

```go
func MyHandler(w http.ResponseWriter, req *http.Request) {
    response := respond.To(w, req)

    fileID := param(req, "file")
    file, ok := fileStore.GetFileInfo(fileID)
    if !ok {
        response.NotFound("file not found: %s", fileID)
        return
    }
    response.RedirectTo(file)
}
```

```go
type S3FileInfo struct {
    Bucket    string
    Directory string
    Name      string
}

func (f S3FileInfo) Redirect() string {
    return fmt.Sprintf("https://%s.s3.amazonaws.com/%s/%s", 
        file.Bucket,
        file.Directory,
        file.Name,
    )
}
```

As you can see, your handler is a bit cleaner, and it's easier to
reason about what it's actually doing. Additionally, you can write
URL formatting tests independent of your handler tests.

One final note. Both `Redirect()` and `RedirectTo()` have "permanent"
variants that result in a 308 HTTP status rather than a 307:
`RedirectPermanent()` and `RedirectPermanentTo()`.

### Responding With Images And Other Raw Files

You can use the `Serve()` and `Download()` functions to deliver
raw file data rather than marshaled JSON. Both accept an `io.Reader`
as the data source.

```go
func ProfilePictureHandler(w http.ResponseWriter, req *http.Request) {
    response := respond.To(w, req)

    userID := param(req, "user")
    image, err := pictureStore.ReadFile(userID + ".jpg")
    defer imageReader.Close()

    // Deliver the picture "inline" for use in an <img> tag.
    response.Serve("profile.jpg", image, err)

    // OR

    // Offer a download dialog for the image
    response.Download("profile.jpg", image, err)
}
```

In addition to writing the bytes, `respond` will apply the correct
`Content-Type` and `Content-Disposition` headers based on the name/extension
of the file you provide.

### Raw Files By Implementing ContentReader

If you'd like to decouple yourself further from the `respond`
library when serving up raw files, you can continue to respond
using `Ok()` with your own structs/values as long as it implements
`ContentReader` - which basically means that it has a `Content()` method
that returns an `io.Reader` with the raw data.

Instead of marshaling the result value as JSON and responding
with those bytes, it will respond with the raw bytes your
reader supplies.

```go
func ExportCSV(w http.ResponseWriter, req *http.Request) {
    // This is an *Export which implements ContentReader
    export := crunchTheNumbers() 

    // Respond with the raw CSV reader data and the following:
    // Status = 200
    // Content-Type = 'application/octet-stream'
    // Content-Disposition = 'inline'
    // Body = (whatever .Read() gave us)
    respond.To(w, req).Ok(export)
}

type Export struct {
    csvData *bytes.Buffer
}

func (e Export) Content() io.Reader {
    return e.csvData
}
```

Most of the time, however, you probably don't want that generic
content type. Additionally, there may be instances where you'd
rather have the client trigger a download rather than consume
the content inline.

To rectify that, you can implement two optional interfaces to
customize both behaviors:

```go
// Implement this to customize the "Content-Type" header.
type ContentTypeReader interface {
    ContentType() string
}

// Implement this to allow an "attachment" disposition instead.
// The value you return will be the default file name offered to
// the client/user when downloading.
type ContentFileNameReader interface {
    ContentFileName() string
}
```

Updating our example to customize both values, we end up
with the following:

```go
func ExportCSV(w http.ResponseWriter, req *http.Request) {
    // This is an *Export which implements all 3 Content-based interfaces
    export := crunchTheNumbers()

    // Respond with the raw CSV reader data and the following:
    // Status = 200
    // Content-Type = 'text/csv'
    // Content-Disposition = 'attachment; filename="super-important-report.csv"'
    // Body = (whatever .Read() gave us)
    respond.To(w, req).Ok(export)
}

// ---

type Export struct {
    RawData *bytes.Buffer
}

func (e Export) Content() io.Reader {
    return e.csvData
}

func (e Export) ContentType() string {
    return "text/csv" 
}

func (e Export) ContentFileName() string {
    return "super-important-report.csv"
}
```

### Responding With HTML

While most of `respond` was built to support building REST APIs,
you can just as easily respond w/ HTML if your application does
server-side rendering. Using the `HTML()` and `HTMLTemplate()`
functions, you can easily send 200 responses w/ the `Content-Type`
set to `text/html; charset=utf-8`.

You can respond with a pre-rendered block of HTML.

```go
func LoginHandler(w http.ResponseWriter, req *http.Request) {
    response := respond.To(w, req)

    // ... do some work ...
    
    html := "<h1>Hello " + username + "</h1>"
    response.HTML(html)
}
```

Or you can use a standard Go `html/template`. The `Responder` will
evaluate the template for you, and the resulting HTML will be
streamed to the HTTP response.

```go
var loginTemplate := template.Must(template.Parse(`
    <h1>Hello {{ .Username }}</h1>
`))

func LoginHandler(w http.ResponseWriter, req *http.Request) {
    response := respond.To(w, req)

    // ... do some work ...
    
    response.HTMLTemplate(loginTemplate, LoginContext{
        Username: username,
    })
}
```

### FAQs

#### Why Not Just Use Gin/Chi/Echo/Fiber/Buffalo/etc?

If you want to use one of these to help you deal with writing
HTTP responses, you also need to buy into their router, their
request data binding, their middleware, their context objects,
and so forth.

The `respond` package lets you stick to the standard library for
all of your HTTP needs. In doing so, you can still bring your own
router/mux, middleware, binding, etc. `respond` is solely focused
on taking the "stank" out of marshaling data and writing response
headers/data.

#### Do You Support Formats Other Than JSON

Nope. I've toyed with adding content negotiation so that if the
caller is asking for XML they can get it. Realistically, if you
jumped on the Go bandwagon to build HTTP service gateways you're
likely using JSON anyway, so this one is fairly low priority.

#### Do You Support Other Template Engines for `HTMLTemplate()`?

Not directly. If you plan to call `HTMLTemplate()` then you have
to use standard Go templates - which is sufficient for most use
cases.

If you do use another template engine like [Plush](https://github.com/gobuffalo/plush),
you can just invoke its evaluation function yourself and pass
the resulting HTML string to `HTML()`. It's not the most memory
efficient, but it's probably more than sufficient for most workloads.


```go
responder := respond.To(w, req)

ctx := plush.NewContext()
ctx.Set("username", "BobLoblaw")

html, err := plush.Render(loginTemplate, ctx)
responder.HTML(html, err)
```

You can even tighten this up more since Go will pass along the
multiple return values properly:

```go
responder := respond.To(w, req)

ctx := plush.NewContext()
ctx.Set("username", "BobLoblaw")

responder.HTML(plush.Render(loginTemplate, ctx))
```

#### There's No Function For The HTTP Status Code I Want To Send Back

To keep the library as lean and mean as possible, `respond` only
has helpers for the most commonly used response status codes. If
you absolutely must send an "I'm a Teapot" response, use the
`Reply()` function.

```go
responder := respond.To(w, req)
...
responder.Reply(http.StatusTeapot, "I'm a teapot")
```
