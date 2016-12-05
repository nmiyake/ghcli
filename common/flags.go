// Copyright 2016 Nick Miyake. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package common

import (
	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/palantir/pkg/cli"
	"github.com/palantir/pkg/cli/flag"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/nmiyake/ghcli/repository"
)

const (
	GitHubTokenFlagName     = "github-token"
	CopyrightAuthorFlagName = "author"
	cacheDirFlagName        = "cache-dir"
	organizationFlagName    = "organization"
	userFlagName            = "user"
)

var (
	GitHubTokenFlag = flag.StringFlag{
		Name:  GitHubTokenFlagName,
		Usage: "GitHub OAuth token for API calls",
	}
	CopyrightAuthorFlag = flag.StringFlag{
		Name:  CopyrightAuthorFlagName,
		Usage: "name of the author/copyright holder to use in licenses that require it",
	}
	cacheDirFlag = flag.StringFlag{
		Name:  cacheDirFlagName,
		Usage: "directory in which to cache GitHub API responses (if absent, an in-memory cache is used instead)",
	}
	organizationFlag = flag.StringFlag{
		Name:  organizationFlagName,
		Usage: "GitHub organization for which repositories are resolved",
	}
	userFlag = flag.StringFlag{
		Name:  userFlagName,
		Usage: "GitHub user for which repositories are resolved",
	}
	AllFlags = []flag.Flag{
		GitHubTokenFlag,
		cacheDirFlag,
		organizationFlag,
		userFlag,
		CopyrightAuthorFlag,
	}
	RepositoryFlags = []flag.Flag{
		GitHubTokenFlag,
		cacheDirFlag,
		organizationFlag,
		userFlag,
	}
)

type GitHubParams interface {
	Token() string
	CacheDir() string
	CachingOAuthGitHubClient() *github.Client
}

type gitHubParams struct {
	token    string
	cacheDir string
}

func (p *gitHubParams) Token() string {
	return p.token
}

func (p *gitHubParams) CacheDir() string {
	return p.cacheDir
}

type GitHubRepositoryParams interface {
	GitHubParams
	RepositoryKey() string
	ProcessRepos(client *github.Client, repos []string, f repository.ProcessFunc) error
}

type gitHubRepositoryParams struct {
	GitHubParams
	user         string
	organization string
}

func (p *gitHubRepositoryParams) RepositoryKey() string {
	if p.organization != "" {
		return p.organization
	}
	return p.user
}

func (p *gitHubRepositoryParams) ProcessRepos(client *github.Client, repos []string, f repository.ProcessFunc) error {
	// if provided list of repos is empty, process all
	if len(repos) == 0 {
		var err error
		if p.organization != "" {
			err = repository.ProcessOrgRepos(client, p.organization, f)
		} else {
			err = repository.ProcessUserRepos(client, p.user, f)
		}
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve repositories")
		}
		return nil
	}

	// otherwise, process provided repositories
	for i, currRepo := range repos {
		repo, _, err := client.Repositories.Get(p.RepositoryKey(), currRepo)
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve repository %s for %s", currRepo, p.RepositoryKey())
		}
		if err := f(repo, repository.Progress{
			CurrPageRepo:     i,
			CurrPageNumRepos: len(repos),
		}); err != nil {
			return err
		}
	}
	return nil
}

func NewGitHubParams(ctx cli.Context) GitHubParams {
	var cacheDir string
	if ctx.Has(cacheDirFlagName) {
		cacheDir = ctx.String(cacheDirFlagName)
	}
	return &gitHubParams{
		token:    ctx.String(GitHubTokenFlagName),
		cacheDir: cacheDir,
	}
}

func NewGitHubRepositoryParams(ctx cli.Context) (GitHubRepositoryParams, error) {
	if ctx.String(userFlagName) == "" && ctx.String(organizationFlagName) == "" {
		return nil, errors.Errorf("either user or organization must be provided")
	} else if ctx.String(userFlagName) != "" && ctx.String(organizationFlagName) != "" {
		return nil, errors.Errorf("user and organization cannot both be provided")
	}

	return &gitHubRepositoryParams{
		GitHubParams: NewGitHubParams(ctx),
		user:         ctx.String(userFlagName),
		organization: ctx.String(organizationFlagName),
	}, nil
}

func (p *gitHubParams) CachingOAuthGitHubClient() *github.Client {
	return CachingOAuthGitHubClient(p.token, p.cacheDir)
}

func CachingOAuthGitHubClient(token, cacheDir string) *github.Client {
	var cache httpcache.Cache
	if cacheDir != "" {
		cache = diskcache.New(cacheDir)
	} else {
		cache = httpcache.NewMemoryCache()
	}
	cachedTransport := httpcache.NewTransport(cache)

	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(oauth2.NoContext, ts)
		cachedTransport.Transport = tc.Transport
	}
	return github.NewClient(cachedTransport.Client())
}
