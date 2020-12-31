# Respond

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
go get -u github.com/robsignorelli/respond
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
// Responds w/ a 307-style redirect to the given URL
response.Redirect("https://google.com?q=", searchText)
// Responds w/ a 308-style redirect to the given URL
response.RedirectPermanent("https://google.com?q=", searchText)
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
    respond.Serve("profile.jpg", image, err)

    // OR

    // Offer a download dialog for the image
    respond.Download("profile.jpg", image, err)
}
```

In addition to writing the bytes, `respond` will apply the correct
`Content-Type` and `Content-Disposition` headers based on the name/extension
of the file you provide.

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

#### Can I Send HTML Back?

That's on my "up-next" list of things to tackle. You could hack
it right now by using the `ServeBytes("index.html", "... some HTML ...")`,
but I realize that's clunky as hell. I plan to add first-class support
so that you can either feed raw HTML strings or utilize the
`template/html` package for dynamic HTML. I started with my own
needs for a REST API and that's the next logical step.
