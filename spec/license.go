// Copyright 2016 Nick Miyake. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package spec

import (
	"fmt"
	"io"
	"strings"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"

	"github.com/nmiyake/ghcli/license"
	"github.com/nmiyake/ghcli/repository"
)

type licenseAnalyzer struct {
	client     *github.Client
	authorName string
	cache      license.Cache
}

func NewLicenseAnalyzer(client *github.Client, authorName string) Analyzer {
	return &licenseAnalyzer{
		client:     client,
		authorName: authorName,
		cache:      license.NewCache(client),
	}
}

func (d *licenseAnalyzer) Name() string {
	return "license"
}

func (d *licenseAnalyzer) Diff(def repository.Definition, info repository.Info) string {
	if def.License == "custom" {
		// custom license -- assume correct
		return ""
	}
	wantLicenseType := strings.TrimPrefix(def.License, "custom-")
	var gotLicense string
	if info.License != nil && info.License.SPDXID != nil {
		gotLicense = *info.License.SPDXID
	}
	if wantLicenseType == "" && gotLicense == "" {
		return ""
	}
	if wantLicenseType != gotLicense {
		// detected license type is different from specification
		return stringDiff("license type", wantLicenseType, gotLicense)
	}
	if _, err := license.VerifyRepositoryLicenseCorrect(info.RepoLicense, &info.Repository, d.authorName, d.cache); license.IsIncorrect(err) {
		// content of license differs from expectation
		return joinDiff(fmt.Sprintf("%s content (%s)", *info.RepoLicense.Path, *info.RepoLicense.License.Name), strings.Split(license.Diff(err), "\n")...)
	}
	return ""
}

func (d *licenseAnalyzer) CanFix() bool {
	return d.client != nil && d.cache != nil
}

func (d *licenseAnalyzer) Fix(def repository.Definition, info repository.Info, stdout io.Writer) error {
	prParams := license.DefaultPRParams("")
	prParams.Body = "Fix license for repository to match specification."
	if err := license.ApplyStandard(d.client, info, strings.TrimPrefix(def.License, "custom-"), d.authorName, prParams, d.cache, stdout); err != nil {
		return errors.Wrapf(err, "failed to fix license")
	}
	return nil
}
