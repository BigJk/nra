// Package nra (not REST again) provides a minimal way to make your go functions
// callable from Javascript.
package nra

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// Bind creates a http.HandlerFunc from a function.
// this handler can than be called from Javascript.
//
// The fn function can take any number of arguments,
// but needs to return either 1 or 2 values.
//
// 2 values:
// The first one is your custom return type (can also be interface{})
// and the second one must be a error.
//
// 1 value:
// The return must be a error.
//
// a valid function would be:
//   func CallMe(a int, b string) (string, error) {
//     if(a == 0) {
//       return "", fmt.Errorf("something went wrong")
//     }
//     return "hello world", nil
//   }
//
func Bind(fn interface{}) (http.HandlerFunc, error) {
	// get the type and value via reflection.
	fnType := reflect.TypeOf(fn)
	fnValue := reflect.ValueOf(fn)

	// check if fn is a function.
	if fnType.Kind() != reflect.Func {
		return nil, errors.New("fn wasn't a function")
	}

	// check that fn has a single or two returns.
	if fnType.NumOut() == 0 || fnType.NumOut() > 2 {
		return nil, errors.New("fn doesn't return 1 or 2 values")
	}

	errReturnIndex := 1
	if fnType.NumOut() == 1 {
		errReturnIndex = 0
	}

	// check if the expected error return value implements the error interface.
	if fnType.Out(errReturnIndex).Kind() != reflect.Interface || !fnType.Out(errReturnIndex).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, errors.New("fn doesn't return a error as second value")
	}

	passRequest := false
	argNum := fnType.NumIn()
	argOffset := 0

	// Check if *http.Request should be passed to target function.
	if argNum > 0 && fnType.In(0) == reflect.TypeOf(new(http.Request)) {
		passRequest = true
		argNum--
		argOffset++
	}

	return func(writer http.ResponseWriter, request *http.Request) {
		// nra only accepts POST requests because it
		// will get the arguments to call fn from the
		// post data.
		if request.Method != "POST" {
			http.Error(writer, "\"only POST requests are permitted\"", http.StatusBadRequest)
			return
		}

		// on the Javascript side the arguments will
		// be encoded as a array that contains variable types.
		// So we just generically decode it into a []interface{}.
		// first.
		var args []interface{}
		if err := json.NewDecoder(request.Body).Decode(&args); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		if err := request.Body.Close(); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		// check if number of arguments match the fn function.
		if len(args) != argNum {
			http.Error(writer, "\"number of arguments mismatch\"", http.StatusBadRequest)
			return
		}

		// now we need to check each argument if it
		// matches the argument of the fn function, or
		// can be dynamically converted to the right type.
		var callValues []reflect.Value
		for i := range args {
			argType := reflect.TypeOf(args[i])

			// check if the argument was null on the javascript side.
			if argType == nil {
				// check if the argument in fn can be nil. if it
				// can be we will create a nil value for the type.
				switch fnType.In(i + argOffset).Kind() {
				case reflect.Ptr:
					fallthrough
				case reflect.Uintptr:
					fallthrough
				case reflect.Map:
					fallthrough
				case reflect.Array:
					fallthrough
				case reflect.Slice:
					callValues = append(callValues, reflect.New(fnType.In(i+argOffset)).Elem())
					continue
				}

				// otherwise we return a error because the argument couldn't
				// be a nil value.
				http.Error(writer, fmt.Sprintf("\"%d. can't be null\"", i+1), http.StatusBadRequest)
				return
			}

			// if our target argument of the fn function is a struct and
			// the argument on the javascript side was a object the decoded
			// argument will always be the type map[string]interface{}.
			//
			// we can dynamically create the struct we want and decode the
			// map[string]interface{} to the struct with the help of the
			// mapstructure package.
			//
			// same works with converting a javascript array to a golang
			// slice.
			if fnType.In(i+argOffset).Kind() == reflect.Struct && argType.Kind() == reflect.Map || fnType.In(i+argOffset).Kind() == reflect.Slice && argType.Kind() == reflect.Slice {
				s := reflect.New(fnType.In(i + argOffset))

				// Create a decoder that honors the json tags
				decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
					Metadata: nil,
					TagName:  "json",
					Result:   s.Interface(),
				})

				if err != nil {
					http.Error(writer, fmt.Sprintf("\"error while creating decoder: %v\"", err), http.StatusBadRequest)
				}

				if err := decoder.Decode(args[i]); err != nil {
					http.Error(writer, err.Error(), http.StatusBadRequest)
					return
				}

				callValues = append(callValues, s.Elem())
				continue
			}

			// check if the argument types mismatch.
			if fnType.In(i+argOffset).Kind() != argType.Kind() {
				// numbers that are generically decoded from JSON will
				// always be float64. In case fn wants some other number
				// type we can just convert it to the target type.
				if argType.Kind() == reflect.Float64 {
					switch fnType.In(i + argOffset).Kind() {
					case reflect.Int:
						fallthrough
					case reflect.Int8:
						fallthrough
					case reflect.Int16:
						fallthrough
					case reflect.Int32:
						fallthrough
					case reflect.Int64:
						fallthrough
					case reflect.Uint8:
						fallthrough
					case reflect.Uint16:
						fallthrough
					case reflect.Uint32:
						fallthrough
					case reflect.Uint64:
						fallthrough
					case reflect.Float32:
						callValues = append(callValues, reflect.ValueOf(args[i]).Convert(fnType.In(i+argOffset)))
						continue
					}
				}

				// otherwise we return a error as no conversion was applicable.
				http.Error(writer, fmt.Sprintf("\"mismatching argument type of %d. argument. got=%s expected=%s\"", i+1, argType.Kind().String(), fnType.In(i+argOffset).Kind().String()), http.StatusBadRequest)
				return
			}

			// otherwise the arguments have the same type so no conversion is needed.
			callValues = append(callValues, reflect.ValueOf(args[i]))
		}

		// call our fn function with the collected values.
		var res []reflect.Value
		if passRequest {
			res = fnValue.Call(append([]reflect.Value{reflect.ValueOf(request)}, callValues...))
		} else {
			res = fnValue.Call(callValues)
		}

		// check if error is present and return it.
		if res[errReturnIndex].Interface() != nil {
			err := res[errReturnIndex].Interface().(error)
			if err != nil {
				http.Error(writer, fmt.Sprintf("\"%s\"", err.Error()), http.StatusBadRequest)
				return
			}
		}

		// if the functions has a return value besides the error
		// JSON encode the returned value and write it to the response.
		if errReturnIndex == 1 {
			_ = json.NewEncoder(writer).Encode(res[0].Interface())
		}
	}, nil
}

// MustBind is the same as Bind but can't return a error.
// this can be used if you want to directly pass the result
// to http.HandleFunc.
func MustBind(fn interface{}) http.HandlerFunc {
	h, err := Bind(fn)
	if err != nil {
		panic("nra: bind failed with: " + err.Error())
	}
	return h
}
