package repository_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nmiyake/ghcli/repository"
)

const testYML = `
- name: octocat/Hello-World
  description: "Example repo"
  owners: [nmiyake]
  license: "mit"
  patents: true`

func TestGetDefinitionsFromFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/octocat/Hello-World/contents/definitions.yml", r.URL.String())

		content := base64.StdEncoding.EncodeToString([]byte(testYML))
		resp := github.RepositoryContent{
			Content: &content,
		}
		json, err := json.Marshal(resp)
		require.NoError(t, err)

		_, err = w.Write(json)
		require.NoError(t, err)
	}))
	defer ts.Close()

	client := github.NewClient(nil)
	client.BaseURL, _ = url.Parse(ts.URL + "/")

	repo := &github.Repository{
		Owner: &github.User{
			Name: github.String("octocat"),
		},
		Name: github.String("Hello-World"),
	}

	got, err := repository.GetDefinitionsFromFile(client, repo, "definitions.yml")
	require.NoError(t, err)

	want := []repository.Definition{
		{
			FullName:    "octocat/Hello-World",
			Description: "Example repo",
			Owners:      []string{"nmiyake"},
			License:     "mit",
			HasPatents:  true,
		},
	}
	assert.Equal(t, want, got)
}
