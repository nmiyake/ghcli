// Copyright 2016 Nick Miyake. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package license

import (
	"fmt"
	"io"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"

	"github.com/nmiyake/ghcli/repository"
)

type PRParams struct {
	Branch string
	Title  string
	Body   string
}

func DefaultPRParams(licenseName string) PRParams {
	return PRParams{
		Branch: "cli-update-license",
		Title:  "Update LICENSE",
		Body:   fmt.Sprintf("Use standard version of %s.", licenseName),
	}
}

// ApplyStandard applies the standard license of the specified type to the specified repository. Calls Create to get the
// content of the license and calls Apply to apply that license to the repository. copyrightAuthor is used as the author
// name for the license if it uses an author template. If the license uses an author template, then the creation year of
// the repository is used as the creation year and the last update year of the repository is used as the update year.
// See documentation for the Apply function for further information.
func ApplyStandard(client *github.Client, repo repository.Info, licenseType, copyrightAuthor string, prParams PRParams, cache Cache, stdout io.Writer) error {
	wantLicenseContent, err := Create(licenseType, cache, NewAuthorInfo(copyrightAuthor, repo.CreatedAt.Time.Year(), repo.UpdatedAt.Time.Year()))
	if err != nil {
		return err
	}
	return Apply(client, repo, wantLicenseContent, prParams, stdout)
}

// Apply applies the provided license content to the repository by opening a PR on the repository to modify the content
// of the existing license file to be the provided content. If the currently authenticated user has push permissions to
// the repository, a PR is created directly on the repository, otherwise, a PR is created on a fork of the repository
// (and a fork is created if it does not already exist). prParams is used to specify the behavior of how the PR is
// created (branch name, commit title, commit body, etc.).
func Apply(client *github.Client, repo repository.Info, licenseContent string, prParams PRParams, stdout io.Writer) error {
	defaultBranch, _, err := client.Repositories.GetBranch(*repo.Owner.Login, *repo.Name, *repo.DefaultBranch)
	if err != nil {
		return errors.Wrapf(err, "failed to get default branch for %s", *repo.Name)
	}

	latestCommit, _, err := client.Git.GetCommit(*repo.Owner.Login, *repo.Name, *defaultBranch.Commit.SHA)
	if err != nil {
		return errors.Wrapf(err, "failed to get latest commit for branch %s", *defaultBranch.Name)
	}

	// get version of repository for which push permissions are enabled (original repo or fork)
	var prRepo *github.Repository
	if (*repo.Permissions)["push"] {
		fmt.Fprintf(stdout, "User has push permissions to repository\n")
		prRepo = &repo.Repository
	} else {
		// user cannot push to repository -- find or create fork
		if userForkRepo, err := repository.GetUserFork(client, &repo.Repository); err != nil {
			return errors.Wrapf(err, "failed to get fork of repository %s for current authenticated user", *repo.Name)
		} else if userForkRepo != nil {
			// user fork of desired directory already exists -- use it
			fmt.Fprintf(stdout, "User does not have push permissions to repository, but has an existing fork\n")
			prRepo = userForkRepo
		} else {
			// user fork of desired repository does not exist -- create it
			fmt.Fprintf(stdout, "User does not have push permissions to repository and does not have an existing fork\n")

			fmt.Fprintf(stdout, "Forking repository...")
			newForkedRepo, err := repository.CreateFork(client, &repo.Repository, 0)
			if err != nil {
				return errors.Wrapf(err, "failed to create fork of repository %s for current authenticated user", *repo.Name)
			}
			prRepo = newForkedRepo
			fmt.Fprintf(stdout, "OK\n")
		}
	}

	fmt.Fprintf(stdout, "Creating tree...")
	createdTree, _, err := client.Git.CreateTree(*prRepo.Owner.Login, *prRepo.Name, *latestCommit.Tree.SHA, []github.TreeEntry{
		{
			Path:    repo.RepoLicense.Path,
			Mode:    github.String("100644"),
			Type:    github.String("blob"),
			Content: github.String(licenseContent),
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create tree")
	}
	fmt.Fprintf(stdout, "OK\n")

	fmt.Fprintf(stdout, "Creating commit...")
	createdCommit, _, err := client.Git.CreateCommit(*prRepo.Owner.Login, *prRepo.Name, &github.Commit{
		Message: github.String("Update license"),
		Parents: []github.Commit{
			*defaultBranch.Commit,
		},
		Tree: createdTree,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create commit")
	}
	fmt.Fprintf(stdout, "OK\n")

	fmt.Fprintf(stdout, "Creating branch...")
	_, _, err = client.Git.CreateRef(*prRepo.Owner.Login, *prRepo.Name, &github.Reference{
		Ref: github.String("refs/heads/" + prParams.Branch),
		Object: &github.GitObject{
			SHA: createdCommit.SHA,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create reference")
	}
	fmt.Fprintf(stdout, "OK\n")

	prBranchName := prParams.Branch
	if *repo.ID != *prRepo.ID {
		// if repo is a fork, prepend branch name with "username:"
		prBranchName = *prRepo.Owner.Login + ":" + prBranchName
	}

	fmt.Fprintf(stdout, "Creating pull request...")
	_, _, err = client.PullRequests.Create(*repo.Owner.Login, *repo.Name, &github.NewPullRequest{
		Title: github.String(prParams.Title),
		Body:  github.String(prParams.Body),
		Head:  &prBranchName,
		Base:  defaultBranch.Name,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create PR")
	}
	fmt.Fprintf(stdout, "OK\n")
	return nil
}
