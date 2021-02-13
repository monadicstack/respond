package respond_test

import (
	"errors"
	"net/http"

	"github.com/monadicstack/respond"
)

var ErrNotFound = errors.New("not found")
var ErrBadCredentials = errors.New("bad credentials")
var ErrAuthorization = errors.New("authorization")

func param(req *http.Request, name string) string {
	return ""
}

func findUser(id string) (map[string]interface{}, bool) {
	return nil, true
}

func doTask() (interface{}, error) {
	return nil, nil
}

func authenticate(req *http.Request) (*http.Request, error) {
	return nil, nil
}

func authorize(req *http.Request) (*http.Request, error) {
	return nil, nil
}

// A simple example. Format the string "Hello World" as JSON and respond
// to the caller with a 200 status code. We create a "Responder" for this
// request and then use it to respond with that result.
func ExampleTo_basic() {
	_ = func(w http.ResponseWriter, req *http.Request) {
		respond.To(w, req).Ok("Hello World")
	}
}

// You can return any primitive or complex value to the caller as JSON and
// respond with a 200. You focus on your business logic and responders offload
// the burden of marshalling, setting status code, and basic header application.
func ExampleTo_businessLogic() {
	_ = func(w http.ResponseWriter, req *http.Request) {
		response := respond.To(w, req)

		userId := param(req, "user")
		user, ok := findUser(userId)
		if !ok {
			response.NotFound("user not found: %s", userId)
			return
		}
		response.Ok(user)
	}
}

// There are responder functions to support most of the common HTTP failures you typically
// encounter when building APIs/Services.
func ExampleTo_error() {
	_ = func(w http.ResponseWriter, req *http.Request) {
		response := respond.To(w, req)

		result, err := doTask()
		switch err {
		case nil:
			response.Ok(result)
		case ErrNotFound:
			response.NotFound("task not found")
		case ErrBadCredentials:
			response.Unauthorized("bad username/password")
		case ErrAuthorization:
			response.Forbidden("you don't have rights to do that")
		default:
			response.InternalServerError("unexpected error: %v", err)
		}
	}
}

// Success functions like Ok(), Accept(), etc optionally take an error (admittedly by
// bastardizing variadic arguments). If the error you pass is nil, the request will
// result in the 2XX status that you wanted w/ the return value. If the error is NOT
// nil then we'll respond with a 4XX/5XX status and the error's message instead.
//
// We will attempt to assign a meaningful HTTP status code by unwrapping your non-nil
// error and look for one of three functions:
//
// * Status() int
// * StatusCode() int
// * Code() int
//
// We'll assume that the resulting int is the HTTP status code you want to fail with.
func ExampleTo_errorShorthand() {
	_ = func(w http.ResponseWriter, req *http.Request) {
		response := respond.To(w, req)

		// Option 1: pass the values separately
		result, err := doTask()
		response.Ok(result, err)

		// Option 2: pass the return values directly to your responder function
		response.Ok(doTask())
	}
}

// You can use respond with any standard middleware you want.
func ExampleTo_middleware() {
	_ = func(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		response := respond.To(w, req)

		req, err := authenticate(req)
		if err != nil {
			response.Unauthorized(err.Error())
			return
		}

		next(w, req)
	}
}
