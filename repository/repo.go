package repository

import (
	"fmt"
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

type ProcessFunc func(repo *github.Repository, progress Progress) error

type Progress struct {
	CurrPageRepo     int
	CurrPageNumRepos int
	CurrPage         int
	TotalPages       int
}

func (p Progress) String() string {
	msg := fmt.Sprintf("%d/%d", p.CurrPageRepo+1, p.CurrPageNumRepos)
	if p.TotalPages > 1 {
		msg += fmt.Sprintf(", page %d/%d", p.CurrPage, p.TotalPages)
	} else if p.CurrPage > 1 {
		// edge case -- when page is the last page, TotalPages is no longer set to be the total number of pages.
		// Use different logic to show progress in this case.
		msg += fmt.Sprintf(", page %d/%d", p.CurrPage, p.CurrPage)
	}
	return msg
}

// ProcessOrgRepos runs the provided function for every listed repository for the provided organization. If the
// processing function returns an error, the error is returned immediately and all further processing is stopped. If
// batch error processing is desired, the provided function should record and manage the errors itself but return nil as
// an error.
func ProcessOrgRepos(client *github.Client, org string, f ProcessFunc) error {
	return processRepos(func(options github.ListOptions) ([]*github.Repository, *github.Response, error) {
		return client.Repositories.ListByOrg(org, &github.RepositoryListByOrgOptions{ListOptions: options})
	}, f)
}

// ProcessUserRepos runs the provided function for every listed repository for the provided user. If the processing
// function returns an error, the error is returned immediately and all further processing is stopped. If batch error
// processing is desired, the provided function should record and manage the errors itself but return nil as an error.
func ProcessUserRepos(client *github.Client, user string, f ProcessFunc) error {
	return processRepos(func(options github.ListOptions) ([]*github.Repository, *github.Response, error) {
		return client.Repositories.List(user, &github.RepositoryListOptions{ListOptions: options})
	}, f)
}

func processRepos(listFunc func(options github.ListOptions) ([]*github.Repository, *github.Response, error), f ProcessFunc) error {
	hasNext := true
	page := 1
	for hasNext {
		repos, response, err := listFunc(github.ListOptions{
			Page: page,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve repositories")
		}
		for i, repo := range repos {
			if err := f(repo, Progress{
				CurrPageRepo:     i,
				CurrPageNumRepos: len(repos),
				CurrPage:         page,
				TotalPages:       response.LastPage,
			}); err != nil {
				return err
			}
		}
		hasNext = response.NextPage != 0
		page = response.NextPage
	}
	return nil
}

// ProcessCollaborators runs the provided function for every collaborator of the specified repository. If the processing
// function returns an error, the error is returned immediately and all further processing is stopped. If batch error
// processing is desired, the provided function should record and manage the errors itself but return nil as an error.
// If an error occurs due to the GitHub API call failing, the HTTP response is returned as well.
func ProcessCollaborators(client *github.Client, repo *github.Repository, f func(user *github.User) error) (*github.Response, error) {
	hasNext := true
	page := 1
	for hasNext {
		users, response, err := client.Repositories.ListCollaborators(*repo.Owner.Login, *repo.Name, &github.ListOptions{
			Page: page,
		})
		if err != nil {
			return response, errors.Wrapf(err, "failed to retrieve collaborators")
		}
		for _, user := range users {
			if err := f(user); err != nil {
				return nil, err
			}
		}
		hasNext = response.NextPage != 0
		page = response.NextPage
	}
	return nil, nil
}

// GetUserFork returns the a repository owned by the currently authenticated user that is a fork of the provided
// repository. Returns nil if no such repository exists.
func GetUserFork(client *github.Client, repo *github.Repository) (*github.Repository, error) {
	for currRepoPage := 1; currRepoPage != 0; {
		currUserRepos, reposResp, err := client.Repositories.List("", &github.RepositoryListOptions{
			Affiliation: "owner",
			ListOptions: github.ListOptions{
				Page: currRepoPage,
			},
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get repositories for authenticated user")
		}

		for _, currUserRepo := range currUserRepos {
			if *currUserRepo.Fork {
				currRepo, _, err := client.Repositories.GetByID(*currUserRepo.ID)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get repository with ID %d", *currUserRepo.ID)
				}
				if *currRepo.Source.ID == *repo.ID {
					return currRepo, nil
				}
			}
		}
		currRepoPage = reposResp.NextPage
	}
	return nil, nil
}

// CreateFork creates a fork of the given GitHub repository and blocks until the new forked repository is available.
// Waits for a maximum of timeout seconds. If timeout is not positive, it defaults to 60.
func CreateFork(client *github.Client, sourceRepo *github.Repository, timeout int) (*github.Repository, error) {
	fork, _, err := client.Repositories.CreateFork(*sourceRepo.Owner.Login, *sourceRepo.Name, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create fork")
	}

	// creating a fork is asynchronous, so wait until operation has completed
	if timeout <= 0 {
		// if total timeout is not specified, default to 1 minute
		timeout = 60
	}
	success := make(chan bool, 1)
	go func() {
		totalTimeWaited := 0
		currWaitLen := 1
		for totalTimeWaited < timeout {
			if defaultBranch, _, err := client.Repositories.GetBranch(*fork.Owner.Login, *fork.Name, *fork.DefaultBranch); err == nil {
				if _, _, err := client.Git.GetCommit(*fork.Owner.Login, *fork.Name, *defaultBranch.Commit.SHA); err == nil {
					// if commit can be retrieved, fork is ready
					success <- true
					return
				}
			}

			if currWaitLen+totalTimeWaited > timeout {
				// if currWaitLength would cause wait time to exceed total timeout, adjust it
				currWaitLen = timeout - totalTimeWaited
			}

			time.Sleep(time.Duration(currWaitLen) * time.Second)

			// exponential back-off
			totalTimeWaited += currWaitLen
			currWaitLen *= 2
		}
		// if repo fork was not completed
		success <- false
	}()
	if !<-success {
		return nil, errors.Errorf("timed out after waiting %d seconds for fork to be created", timeout)
	}

	return fork, nil
}
