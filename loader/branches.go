package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"golang.org/x/oauth2/clientcredentials"
	"strings"
)

type branch struct {
	Name string `json:"name"`
}

type branchesCollection struct {
	Next     string   `json:"next"`
	Branches []branch `json:"values"`
}

type BranchLoaderInterface interface {
	LoadBranches(bitbucket Bitbucket) ([]string, error)
}

type BranchLoader struct {
}

func (b *BranchLoader) LoadBranches(bitbucket Bitbucket) ([]string, error) {
	ctx := context.Background()
	conf := &clientcredentials.Config{
		ClientID:     bitbucket.ClientID,
		ClientSecret: bitbucket.ClientSecret,
		Scopes:       []string{"repository"},
		TokenURL:     bitbucket.TokenUrl,
	}

	client := conf.Client(ctx)
	resp, err := client.Get(fmt.Sprintf("%s/2.0/repositories/%s/%s/refs/branches?pagelen=100", bitbucket.ApiUrl, bitbucket.Username, bitbucket.RepositoryName))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	branches := []string{}

	if resp.StatusCode == 200 { // OK
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var s = new(branchesCollection)
		err = json.Unmarshal(bodyBytes, &s)
		if err != nil {
			return nil, err
		}

		for _, branch := range s.Branches {
			branches = append(branches, strings.ToLower(branch.Name))
		}
	}

	return branches, nil
}
