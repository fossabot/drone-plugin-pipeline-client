package main

import (
	"testing"

	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"net/http"
)

type TestCase struct {
	Name       string
	Cfg        Config
	AuthHeader string
}

var testCases = []TestCase{
	TestCase{
		Name: "Bearer token auth",
		Cfg: Config{
			Token: "bearertoken",
		},
		AuthHeader: "Bearer bearertoken",
	},
	TestCase{
		Name: "Basic auth",
		Cfg: Config{
			Username: "testing",
		},
		AuthHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("testing:")),
	},
	TestCase{
		Name:       "No auth",
		Cfg:        Config{},
		AuthHeader: "",
	},
	TestCase{
		Name: "Bearer and basic",
		Cfg: Config{
			Username: "testing",
			Token:    "tokenke",
		},
		AuthHeader: "Bearer tokenke",
	},
}

func TestAuthorizationHeader(t *testing.T) {

	for _, tc := range testCases {
		request, _ := http.NewRequest(http.MethodPost, "", nil)
		tc.Cfg.requestAuth(request)
		assert.Equal(t, tc.AuthHeader, request.Header.Get("Authorization"))
	}

}
