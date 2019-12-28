package nra

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	Name     string
	Code     int
	Input    string
	Expected string
	Function interface{}
}

var tests = []testCase{
	{
		Name:     "get_request",
		Input:    "[[10, 11, 12]]",
		Expected: "\"ok\"\n",
		Code:     http.StatusOK,
		Function: func(r *http.Request, a []int) (string, error) {
			if r.Header.Get("TestHeader") != "abc" {
				return "", fmt.Errorf("header was incorrect")
			}
			return "ok", nil
		},
	},
	{
		Name:     "generic",
		Input:    "[1, \"test_string\", 123.2451]",
		Expected: "\"1+test_string+123.2\"\n",
		Code:     http.StatusOK,
		Function: func(a int, b string, c float64) (string, error) {
			return fmt.Sprintf("%d+%s+%.1f", a, b, c), nil
		},
	},
	{
		Name:     "only_error_fail",
		Input:    "[1, \"test_string\", 123.2451]",
		Expected: "\"error\"\n",
		Code:     http.StatusBadRequest,
		Function: func(a int, b string, c float64) error {
			return fmt.Errorf("error")
		},
	},
	{
		Name:     "only_error",
		Input:    "[1, \"test_string\", 123.2451]",
		Expected: "",
		Code:     http.StatusOK,
		Function: func(a int, b string, c float64) error {
			return nil
		},
	},
	{
		Name:     "nil_values",
		Input:    "[null, null, null]",
		Expected: "\"ok\"\n",
		Code:     http.StatusOK,
		Function: func(a *int, b *string, c map[string]string) (string, error) {
			if a == nil && b == nil && c == nil {
				return "ok", nil
			}
			return "", errors.New("not everything was nil")
		},
	},
	{
		Name:     "struct",
		Input:    "[{\"a\":1233,\"b\":{\"c\":\"hello\"}}]",
		Expected: "{\"c\":1233,\"a\":{\"b\":\"hello\"}}\n",
		Code:     http.StatusOK,
		Function: func(a struct {
			A int `json:"c"`
			B struct {
				C string `json:"b"`
			} `json:"a"`
		}) (interface{}, error) {
			return a, nil
		},
	},
	{
		Name:     "not_nilable",
		Input:    "[null, null, null]",
		Expected: "\"1. can't be null\"\n",
		Code:     http.StatusBadRequest,
		Function: func(a int, b string, c float64) (interface{}, error) {
			return nil, nil
		},
	},
	{
		Name:     "wrong_type",
		Input:    "[{\"a\":1233,\"b\":{\"c\":\"hello\"}}]",
		Expected: "\"mismatching argument type of 1. argument. got=map expected=int\"\n",
		Code:     http.StatusBadRequest,
		Function: func(a int) (interface{}, error) {
			return nil, nil
		},
	},
}

func TestBind(t *testing.T) {
	for i := range tests {
		t.Run(tests[i].Name, func(t *testing.T) {
			h, err := Bind(tests[i].Function)
			if !assert.NoError(t, err) {
				return
			}

			req, err := http.NewRequest("POST", "/", bytes.NewBuffer([]byte(tests[i].Input)))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("TestHeader", "abc")

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if !assert.Equal(t, tests[i].Code, rr.Code) || !assert.Equal(t, tests[i].Expected, rr.Body.String()) {
				return
			}
		})
	}
}
