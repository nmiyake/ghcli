package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/google/go-github/github"
	"github.com/palantir/pkg/cli"
	"github.com/palantir/pkg/cli/flag"
	"github.com/pkg/errors"

	"github.com/nmiyake/ghcli/common"
	"github.com/nmiyake/ghcli/license"
	"github.com/nmiyake/ghcli/repository"
)

const reposParamName = "repositories"

var reposParam = flag.StringSlice{
	Name:     reposParamName,
	Usage:    "repositories to process (if unspecified, all repositories are processed)",
	Optional: true,
}

func Verify() cli.Command {
	return cli.Command{
		Name:  "verify",
		Usage: "verify that the license files in repositories have the correct content",
		Flags: append(common.AllFlags,
			reposParam,
		),
		Action: func(ctx cli.Context) error {
			params, err := common.NewGitHubRepositoryParams(ctx)
			if err != nil {
				return err
			}
			return doRepositoryLicense(params, ctx.Slice(reposParamName), ctx.String(common.CopyrightAuthorFlagName), verifyLicenses, false, ctx.App.Stdout)
		},
	}
}

func Fix() cli.Command {
	return cli.Command{
		Name:  "fix",
		Usage: "open PRs to fix license files in repositories that have incorrect content",
		Flags: append(common.AllFlags,
			reposParam,
			common.PromptFlag,
		),
		Action: func(ctx cli.Context) error {
			params, err := common.NewGitHubRepositoryParams(ctx)
			if err != nil {
				return err
			}
			return doRepositoryLicense(params, ctx.Slice(reposParamName), ctx.String(common.CopyrightAuthorFlagName), fixLicenses, ctx.Bool(common.PromptFlagName), ctx.App.Stdout)
		},
	}
}

type processMode int

const (
	verifyLicenses processMode = iota
	fixLicenses
)

func doRepositoryLicense(params common.GitHubRepositoryParams, repos []string, copyrightAuthor string, mode processMode, prompt bool, stdout io.Writer) error {
	client := params.CachingOAuthGitHubClient()
	cache := license.NewCache(client)

	var okRepos []string
	var unableToDetermineRepos []string
	var badRepos []string
	numFixPRsOpened := 0

	f := func(repo *github.Repository, progress repository.Progress) error {
		fmt.Fprintf(stdout, "Verifying license for repository %s (%v)...", *repo.Name, progress)

		repoLicense, err := license.VerifyCorrect(client, repo, copyrightAuthor, cache)
		switch {
		case err == nil:
			okRepos = append(okRepos, *repo.Name)
			fmt.Fprintf(stdout, "OK")
			fmt.Fprintln(stdout)
			return nil
		case license.IsMissing(err):
			msg := fmt.Sprintf("%s: %s", *repo.Name, err.Error())
			unableToDetermineRepos = append(unableToDetermineRepos, msg)
			fmt.Fprintf(stdout, "unable to detect license")
			fmt.Fprintln(stdout)
			return nil
		case license.IsIncorrect(err):
			msg := fmt.Sprintf("%s: %s", *repo.Name, err.Error())
			diff := license.Diff(err)
			if diff != "" {
				msg += ":" + strings.NewReplacer("\n", "\n\t\t").Replace("\n"+diff)
			}
			badRepos = append(badRepos, msg)
			fmt.Fprintf(stdout, "incorrect")
			fmt.Fprintln(stdout)

			if mode != fixLicenses {
				return nil
			}

			repoInfo, err := repository.GetInfo(client, repo)
			if err != nil {
				fmt.Fprintf(stdout, "Failed to get information required to fix repository: %v\n", err)
				return nil
			} else if repoInfo.IsEmpty {
				return errors.Errorf("repository %s is an empty repository", *repo.Name)
			}

			if prompt {
				ok, err := common.Prompt("Open PR for fix", stdout)
				if err != nil {
					return err
				}
				if !ok {
					// if user is given prompt and provides non-"Yes" response, skip
					return nil
				}
			}

			if err := license.ApplyStandard(client, repoInfo, *repoLicense.License.Key, copyrightAuthor, license.DefaultPRParams(*repoLicense.License.Name), cache, stdout); err != nil {
				return err
			}
			numFixPRsOpened++
			return nil
		default:
			fmt.Fprintln(stdout)
			return errors.Wrapf(err, "failed to verify license for repository %s", *repo.Name)
		}
	}

	if len(repos) == 0 {
		if err := params.ProcessRepos(client, f); err != nil {
			return errors.Wrapf(err, "failed to retrieve repositories")
		}
	} else {
		for i, currRepo := range repos {
			repo, _, err := client.Repositories.Get(params.RepositoryKey(), currRepo)
			if err != nil {
				return errors.Wrapf(err, "failed to retrieve repository %s for %s", currRepo, params.RepositoryKey())
			}
			if err := f(repo, repository.Progress{
				CurrPageRepo:     i,
				CurrPageNumRepos: len(repos),
			}); err != nil {
				return err
			}
		}
	}

	if mode == verifyLicenses {
		fmt.Fprintln(stdout, repoMessage(fmt.Sprintf("%s had correct license files", pluralizeRepo(len(okRepos))), okRepos))
		fmt.Fprintln(stdout, repoMessage(fmt.Sprintf("%s had incorrect license files", pluralizeRepo(len(badRepos))), badRepos))
		fmt.Fprintln(stdout, repoMessage(fmt.Sprintf("Unable to determine license type for %s", pluralizeRepo(len(unableToDetermineRepos))), unableToDetermineRepos))
	} else {
		fmt.Fprintf(stdout, "Examined %s and opened %s.\n", pluralizeRepo(len(okRepos)+len(badRepos)+len(unableToDetermineRepos)), pluralizePR(numFixPRsOpened))
	}
	return nil
}

func repoMessage(msg string, repos []string) string {
	if len(repos) == 0 {
		return msg
	}
	return strings.Join(append([]string{msg + ":"}, repos...), "\n\t")
}

func pluralizeRepo(n int) string {
	return pluralize(n, "repository", "repositories")
}

func pluralizePR(n int) string {
	return pluralize(n, "pull request", "pull requests")
}

func pluralize(n int, singular, plural string) string {
	modifier := plural
	if n == 1 {
		modifier = singular
	}
	return fmt.Sprintf("%d %s", n, modifier)
}
