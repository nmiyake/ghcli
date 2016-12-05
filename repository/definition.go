// Copyright 2016 Nick Miyake. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package repository

import (
	"encoding/base64"
	"net/http"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type DefinitionSlice []Definition

func (p DefinitionSlice) Len() int { return len(p) }
func (p DefinitionSlice) Less(i, j int) bool {
	return strings.ToLower(p[i].FullName) < strings.ToLower(p[j].FullName)
}
func (p DefinitionSlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

type CaseInsensitiveStrings []string

func (p CaseInsensitiveStrings) Len() int { return len(p) }
func (p CaseInsensitiveStrings) Less(i, j int) bool {
	return strings.ToLower(p[i]) < strings.ToLower(p[j])
}
func (p CaseInsensitiveStrings) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

type Definition struct {
	FullName    string   `yaml:"name" json:"name"` // full name of the repository, e.g. "octocat/Hello-World"
	Description string   `yaml:"description" json:"description"`
	Owners      []string `yaml:"owners" json:"owners"` // GitHub usernames of owners
	// License is the SPDX identifier for the license intended to be used by the repository. If the repository uses
	// a derivative or custom form of an existing known license, it should be specified as "custom-{{SPDX_ID}}". If
	// the license used by a repository is a custom one that is not based on an existing license with an SPDX ID,
	// the value should be "custom".
	License    string `yaml:"license" json:"license"`
	HasPatents bool   `yaml:"patents" json:"patents"` // true if repository uses patents and should contain a "PATENTS.txt" file
}

type Info struct {
	github.Repository
	RepoLicense *github.RepositoryLicense
	IsEmpty     bool // true if repository is empty
	Owners      []string
	HasPatents  bool
}

func (i *Info) ToDefinition() Definition {
	var description string
	if i.Description != nil {
		description = *i.Description
	}
	var license string
	if i.License != nil && i.License.SPDXID != nil {
		license = *i.License.SPDXID
	}
	return Definition{
		FullName:    *i.FullName,
		Description: description,
		Owners:      i.Owners,
		License:     license,
		HasPatents:  i.HasPatents,
	}
}

// GetInfo returns the Info for the given repo using the provided client.
func GetInfo(client *github.Client, repo *github.Repository) (Info, error) {
	_, resp, err := client.Repositories.ListContributors(*repo.Owner.Login, *repo.Name, nil)
	if err != nil {
		return Info{}, errors.Wrapf(err, "failed to get contributors for %s", *repo.FullName)
	}
	if resp.StatusCode == http.StatusNoContent {
		// if StatusNoContent is returned, repository exists but is empty
		return Info{
			Repository: *repo,
			IsEmpty:    true,
		}, nil
	}

	var repoLicense *github.RepositoryLicense
	if repo.License != nil {
		repoLicense, _, err = client.Repositories.License(*repo.Owner.Login, *repo.Name)
		if err != nil {
			return Info{}, errors.Wrapf(err, "failed to get license for %s", *repo.FullName)
		}
	}

	var owners []string
	if response, err := ProcessCollaborators(client, repo, func(user *github.User) error {
		if (*user.Permissions)["admin"] {
			owners = append(owners, *user.Login)
		}
		return nil
	}); err != nil {
		if response == nil || response.StatusCode != 403 {
			return Info{}, errors.Wrapf(err, "failed to get collaborators for %s", *repo.FullName)
		}
		// if response code is 403, keep owners as nil
	}
	sort.Sort(CaseInsensitiveStrings(owners))

	hasPatents, err := hasPatentsFile(client, repo)
	if err != nil {
		return Info{}, err
	}

	return Info{
		Repository:  *repo,
		RepoLicense: repoLicense,
		Owners:      owners,
		HasPatents:  hasPatents,
	}, nil
}

// Returns true if the provided repository has a "patents" or "patents.txt" file (case-insensitive) at the top level
// (root directory) of the repository.
func hasPatentsFile(client *github.Client, repo *github.Repository) (bool, error) {
	_, dir, _, err := client.Repositories.GetContents(*repo.Owner.Login, *repo.Name, "", nil)
	if err != nil {
		return false, errors.Wrapf(err, "failed to list contents of repository %+v", *repo)
	} else if dir == nil {
		return false, errors.Errorf("failed to list contents of repository %+v", *repo)
	}
	for _, currFile := range dir {
		switch strings.ToLower(*currFile.Name) {
		case "patents", "patents.txt":
			return true, nil
		}
	}
	return false, nil
}

func GetDefinitionsFromFile(client *github.Client, repo *github.Repository, path string) ([]Definition, error) {
	encodedDefContents, _, _, err := client.Repositories.GetContents(*repo.Owner.Name, *repo.Name, path, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get repo definition file")
	}
	if encodedDefContents == nil {
		return nil, errors.Errorf("GetContents for path %s for user %s in repository %s returned nil", path, *repo.Owner.Name, *repo.Name)
	}
	defContents, err := base64.StdEncoding.DecodeString(*encodedDefContents.Content)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode %s from Base64 encoding", encodedDefContents)
	}
	var output []Definition
	if err := yaml.Unmarshal(defContents, &output); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %s as YML", string(defContents))
	}
	return output, nil
}
