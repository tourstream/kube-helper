package util

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBranches(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Basic Q2xpZW50SWQ6Q2xpZW50IFNlY3JldA==", r.Header.Get("Authorization"))
		fmt.Fprintln(w, "{\"access_token\" : \"tolen\"}")
	}))
	defer ts.Close()

	branchesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "{\"values\" : [{\"name\": \"foo\"}]}")
	}))
	defer branchesServer.Close()

	branches, err := GetBranches(Bitbucket{
		ClientID:       "ClientId",
		ClientSecret:   "Client Secret",
		Username:       "Username",
		RepositoryName: "repo",
		TokenUrl:       ts.URL,
		ApiUrl:         branchesServer.URL,
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{"foo"}, branches)

}
