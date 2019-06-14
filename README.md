# nra
[![Documentation](https://godoc.org/github.com/BigJk/nra/console?status.svg)](http://godoc.org/github.com/BigJk/ramen/console) [![Go Report Card](https://goreportcard.com/badge/github.com/BigJk/nra)](https://goreportcard.com/report/github.com/BigJk/nra)

nra (not REST again) is a minimal RPC library to call Go from Javascript. After building yet another REST api just to get some data from a Go backend to a small frontend I grew tired of the process. This library is not intended as a replacement for REST. It purpose is to get data from Go to Javascript with the smallest amount of work. It's perfect for small pet projects where you don't need any advanced features, high security or insane throughput.

# Javascript Code

This code is all you need to call a function that is defined in your Go code. Just copy and paste it into your Javascript.

```Javascript
function call(func, ...args) {
  return new Promise(function(resolve, reject) {
    var request = new XMLHttpRequest();
    request.open('POST', '/rpc/' + func, true);

    request.onload = function() {
      if (request.status === 200) {
        resolve(JSON.parse(request.responseText));
      } else {
        reject(request.responseText);
      }
    };

    request.onerror = function() {
      reject(request.responseText);
    };

    request.send(JSON.stringify(args));
  });
}
```

### Minified

```Javascript
function call(e,...n){return new Promise(function(r,s){var t=new XMLHttpRequest;t.open("POST","/rpc/"+e,!0),t.onload=function(){200===t.status?r(JSON.parse(t.responseText)):s(t.responseText)},t.onerror=function(){s(t.responseText)},t.send(JSON.stringify(n))})}
```

# Example

Imagine you have a Go application that stores logs and you want to build a small Frontend that shows the last 100 logs that contain some search string.

```Go
package main

import (
	"net/http"
	"strings"
  "fmt"

	"github.com/BigJk/nra"
)

func main() {
  // RPC functions should be prefixed
  // with the '/rpc/' path when registering.
  http.HandleFunc("/rpc/get_logs", nra.MustBind(func(search string, limit int) ([]string, error) {
    // lets generate some fake data here.
    // normally some cool database access
    // would happen here.
    fakeLogs := make([]string, limit)
    for i := 0; i < limit; i++ {
      fakeLogs[i] = fmt.Sprintf("%d. some data %s", i, search)
    }
    return fakeLogs, nil
  }))

  // host your html, javascript etc.
  http.Handle("/", http.FileServer(http.Dir("static")))

  // start the server
  http.ListenAndServe(":8765", nil)
}
```

Somewhere in your Javascript

```Javascript
call('get_logs', 'error', 100).then(function(logs) {
  console.log(logs);
}, function(err) {
  console.log(logs);
})
```

Thats it. You can bind any Go function as long as it returns 2 values. The first one can be of any type you like to return to your Javascript and the second one must be a error. Checking if the call from Javascript has the correct arguments and marshaling and unmarshaling the data will all be handled by nra.

# How does it work?

#### Go

Feel free to take a look at ``nra.go`` it's not even 200 lines of code and most of that are comments! To keep it short. nra looks at what arguments your function takes. When a request hits the generated handler function it will unmarshal the body of the request (which will contain the arguments that have been passed to the Javascript ``call`` function) and checks if the arguments and their type match your function. If the arguments match or some valid conversion exists (float64 to int for example) it will call your function with the received arguments. If your function returns no error the resulting value of your function will be JSON encoded and sent as response to the request. If some error occurs the text of the error will be send with the status code ``http.StatusBadRequest``.

#### Javascript

The Javascript part of nra does nothing more than sending a ``POST`` request to ``/rpc/YOUR_FUNCTION_NAME`` with JSON encoded arguments to your backend.
