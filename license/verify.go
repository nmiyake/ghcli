package license

import (
	"encoding/base64"
	"fmt"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/pmezard/go-difflib/difflib"
)

type licenseErrorType int

const (
	errorMissing licenseErrorType = iota
	errorIncorrect
)

type repoLicenseError struct {
	ErrType licenseErrorType
	Message string
	Diff    string
}

func (e *repoLicenseError) Error() string {
	return e.Message
}

func IsMissing(err error) bool {
	if err, ok := err.(*repoLicenseError); ok && err.ErrType == errorMissing {
		return true
	}
	return false
}

func IsIncorrect(err error) bool {
	if err, ok := err.(*repoLicenseError); ok && err.ErrType == errorIncorrect {
		return true
	}
	return false
}

func Diff(err error) string {
	if err, ok := err.(*repoLicenseError); ok {
		return err.Diff
	}
	return ""
}

func VerifyCorrect(client *github.Client, repo *github.Repository, authorName string, cache Cache) (github.RepositoryLicense, error) {
	license, _, err := client.Repositories.License(*repo.Owner.Login, *repo.Name)
	if err != nil {
		msg := "no license detected"
		if *repo.Fork {
			msg = "license cannot be detected for forked repositories (this is a known GitHub API issue)"
		}
		return github.RepositoryLicense{}, &repoLicenseError{ErrType: errorMissing, Message: msg}
	}

	// content of license currently in repository
	actualLicenseBytes, err := base64.StdEncoding.DecodeString(*license.Content)
	if err != nil {
		return *license, errors.Wrapf(err, "failed to decode license content")
	}
	actualLicenseContent := string(actualLicenseBytes)

	expectedLicenseContent, err := Create(*license.License.Key, cache, NewAuthorInfo(authorName, repo.CreatedAt.Time.Year(), repo.UpdatedAt.Time.Year()))
	if err != nil {
		return *license, err
	}

	if actualLicenseContent == expectedLicenseContent {
		// license matches
		return *license, nil
	}

	// license does not match -- determine why the mismatch occurred and provide most appropriate message
	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(expectedLicenseContent),
		B:        difflib.SplitLines(actualLicenseContent),
		FromFile: "Expected",
		ToFile:   "Actual",
		Context:  0,
	})
	if err != nil {
		return github.RepositoryLicense{}, errors.Wrapf(err, "failed to compute diff")
	}

	var msg string
	if spec, ok := licensesMap[*license.License.Key]; ok && hasAuthorInfo(actualLicenseContent) && *license.SHA == spec.SHA1 {
		// spec is known to include author information and the hash of the LICENSE file matches
		// the known hash -- error is that template was not filled out
		msg = fmt.Sprintf("uses unmodified version of %s license (copyright year and author should be filled out)", *license.License.Name)
	} else {
		msg = fmt.Sprintf("actual content of license does not match expected content")
	}
	return *license, &repoLicenseError{ErrType: errorIncorrect, Message: msg, Diff: diff}
}
