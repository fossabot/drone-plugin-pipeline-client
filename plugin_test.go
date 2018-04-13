package main

import (
	"testing"

	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	orgsResponse        = `[{"id":1,"createdAt":"2018-04-11T13:58:55Z","updatedAt":"2018-04-11T13:58:55Z","name":"org1"},{"id":2,"githubId":32848483,"createdAt":"2018-04-11T13:58:55Z","updatedAt":"2018-04-11T13:58:55Z","name":"org2"}]`
	invalidJsonResponse = "invalid json response payload"
)

func TestAuthorizationHeader(t *testing.T) {

	testCases := []struct {
		name       string
		config     Config
		authHeader string
	}{
		{
			name: "Bearer token auth",
			config: Config{
				Token: "bearertoken",
			},
			authHeader: "Bearer bearertoken",
		},
		{
			name: "Basic auth",
			config: Config{
				Username: "testing",
			},
			authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("testing:")),
		},
		{
			name:       "No auth",
			config:     Config{},
			authHeader: "",
		},
		{
			name: "Bearer and basic",
			config: Config{
				Username: "testing",
				Token:    "tokenke",
			},
			authHeader: "Bearer tokenke",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, _ := http.NewRequest(http.MethodPost, "", nil)
			tc.config.requestAuth(request)
			assert.Equal(t, tc.authHeader, request.Header.Get("Authorization"))
		})
	}

}

func TestPlugin_GetOrgId(t *testing.T) {
	tests := []struct {
		name   string                 //name of the test case
		plugin Plugin                 // the plugin (fixture)
		assert func(i int, err error) // assertions
	}{
		{
			name: "org id already retrieved (available in the configuration)",
			plugin: Plugin{
				Config: Config{
					OrgId: 1,
				},
			},
			assert: func(i int, err error) {
				assert.Equal(t, 1, i, "the returned org id is invalid")
				assert.Nil(t, err, "the returned error must be nil")
			},
		},
		{
			name: "retrieving org id - no repo owner found",
			plugin: Plugin{
				ApiCall: func(config *Config, url string, method string, body io.Reader) *http.Response {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte(orgsResponse))),
					}
				},
				Config: Config{},
			},
			assert: func(i int, err error) {
				assert.Equal(t, 0, i, "the returned org id must be 0 (nil value)")
				assert.NotNil(t, err, "the returned error must not be nil")
			},
		},
		{
			name: "retrieving org id - repo owner found",
			plugin: Plugin{
				ApiCall: func(config *Config, url string, method string, body io.Reader) *http.Response {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte(orgsResponse))),
					}
				},
				Repo: Repo{
					Owner: "org1",
				},
				Config: Config{},
			},
			assert: func(i int, err error) {
				assert.Equal(t, 1, i)
				assert.Nil(t, err)
			},
		},
		{
			name: "retrieving org id - not OK response code",
			plugin: Plugin{
				ApiCall: func(config *Config, url string, method string, body io.Reader) *http.Response {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Status:     "200 OK",
						Body:       ioutil.NopCloser(bytes.NewReader([]byte(orgsResponse))),
					}
				},
				Config: Config{},
			},
			assert: func(i int, err error) {
				assert.Equal(t, 0, i, "the returned org id must have the nil value")
				assert.Equal(t, errors.Errorf("could not retrieve organizations. status: [ %s ]", "200 OK").Error(), err.Error(), "unexpected error message")
			},
		},
		{
			name: "retrieving org id - invalid json payload",
			plugin: Plugin{
				ApiCall: func(config *Config, url string, method string, body io.Reader) *http.Response {

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte(invalidJsonResponse))),
					}
				},
				Config: Config{},
			},
			assert: func(i int, err error) {
				assert.Equal(t, 0, i, "the returned org id must have the nil value")
				assert.EqualError(t, err, fmt.Sprintf("could not parse orgs response [ %s ]: invalid character '%s' looking for beginning of value",
					invalidJsonResponse, "i"), "Invalide error message")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.assert(test.plugin.GetOrgId())
		})
	}
}
