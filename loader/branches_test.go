package loader

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBranches(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Basic Q2xpZW50SWQ6Q2xpZW50K1NlY3JldA==", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{\"access_token\" : \"tolen\"}")
	}))
	defer ts.Close()

	branchesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "{\"values\" : [{\"name\": \"foo\"}]}")
	}))
	defer branchesServer.Close()

	branches, err := new(BranchLoader).LoadBranches(Bitbucket{
		ClientID:       "ClientId",
		ClientSecret:   "Client Secret",
		Username:       "Username",
		RepositoryName: "repo",
		TokenURL:       ts.URL,
		APIURL:         branchesServer.URL,
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{"foo"}, branches)

}

func TestGetBranchesWithHttpError(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Basic Q2xpZW50SWQ6Q2xpZW50K1NlY3JldA==", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{\"access_token\" : \"tolen\"}")
	}))
	defer ts.Close()

	branchesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer branchesServer.Close()

	branches, err := new(BranchLoader).LoadBranches(Bitbucket{
		ClientID:       "ClientId",
		ClientSecret:   "Client Secret",
		Username:       "Username",
		RepositoryName: "repo",
		TokenURL:       ts.URL,
		APIURL:         branchesServer.URL,
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{}, branches)

}

func TestGetBranchesWithHttpBodyError(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Basic Q2xpZW50SWQ6Q2xpZW50K1NlY3JldA==", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{\"access_token\" : \"tolen\"}")
	}))
	defer ts.Close()

	branchesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer branchesServer.Close()

	_, err := new(BranchLoader).LoadBranches(Bitbucket{
		ClientID:       "ClientId",
		ClientSecret:   "Client Secret",
		Username:       "Username",
		RepositoryName: "repo",
		TokenURL:       ts.URL,
		APIURL:         branchesServer.URL,
	})
	assert.EqualError(t, err, "unexpected end of JSON input")

}
