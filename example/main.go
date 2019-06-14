package main

import (
	"net/http"
	"strings"

	"github.com/BigJk/nra"
)

// example index page that calls the rpc functions on button press.
var index = `
<!DOCTYPE html>
<html lang="en">
  <title></title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <body>
	<script>
		function call(func, ...args) {
			return new Promise((resolve, reject) => {
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

		function add_example() {
			call('add', 1, 5, 10).then(function(result) {
				alert("Add Result: " + result);
			}, function(err) {
				alert("Error: " + err);
			});
		}

		function echo_example() {
			call('echo', 'double me').then(function(result) {
				alert("Add Result: " + result);
			}, function(err) {
				alert("Error: " + err);
			});
		}
	</script>

	<button onClick="add_example()">1 + 5 + 10</button>
	<button onClick="echo_example()">Hello</button>
  </body>
</html>
`

// some function that you would like to call from javascript
func addFunction(a int, b float64, c uint8) (float64, error) {
	return float64(a) + b + float64(c), nil
}

func main() {
	// binding the function with error checking
	add, err := nra.Bind(addFunction)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/rpc/add", add)

	// binding the function inline
	http.HandleFunc("/rpc/echo", nra.MustBind(func(s string) (string, error) {
		return strings.Repeat(s, 2), nil
	}))

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(index))
	})

	// start the server
	panic(http.ListenAndServe(":8765", nil))
}
