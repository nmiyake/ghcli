package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	"github.com/palantir/pkg/cli"
	"github.com/palantir/pkg/cli/flag"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/nmiyake/ghcli/common"
	"github.com/nmiyake/ghcli/repository"
	"github.com/nmiyake/ghcli/spec"
)

const (
	outputFileParamName = "output"
	specFileParamName   = "spec"
	reposFlagName       = "repositories"
)

var (
	outputFileParam = flag.StringParam{
		Name:  outputFileParamName,
		Usage: "file to which repository specification is written",
	}
	specFileParam = flag.StringParam{
		Name:  specFileParamName,
		Usage: "repository specification file",
	}
	reposFlag = flag.StringFlag{
		Name:  reposFlagName,
		Usage: "repositories to process (if specified, only these repositories are processed)",
	}
)

func CreateSpec() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create GitHub repository specification",
		Flags: append(common.RepositoryFlags,
			reposFlag,
			outputFileParam,
		),
		Action: func(ctx cli.Context) error {
			params, err := common.NewGitHubRepositoryParams(ctx)
			if err != nil {
				return err
			}
			return doCreateSpec(params, getRepos(ctx), ctx.String(outputFileParamName), ctx.App.Stdout)
		},
	}
}

func VerifySpec() cli.Command {
	return cli.Command{
		Name:  "verify",
		Usage: "verify GitHub repository specification",
		Flags: append(common.AllFlags,
			reposFlag,
			specFileParam,
		),
		Action: func(ctx cli.Context) error {
			params, err := common.NewGitHubRepositoryParams(ctx)
			if err != nil {
				return err
			}
			return processSpec(params, getRepos(ctx), ctx.String(specFileParamName), getAnalyzers(params, ctx), verifyMode, true, ctx.App.Stdout)
		},
	}
}

func ApplySpec() cli.Command {
	return cli.Command{
		Name:  "apply",
		Usage: "apply GitHub repository specification",
		Flags: append(common.AllFlags,
			reposFlag,
			specFileParam,
			common.PromptFlag,
		),
		Action: func(ctx cli.Context) error {
			params, err := common.NewGitHubRepositoryParams(ctx)
			if err != nil {
				return err
			}
			return processSpec(params, getRepos(ctx), ctx.String(specFileParamName), getAnalyzers(params, ctx), applyMode, ctx.Bool(common.PromptFlagName), ctx.App.Stdout)
		},
	}
}

func getRepos(ctx cli.Context) []string {
	var repos []string
	if ctx.Has(reposFlagName) {
		repos = strings.Split(ctx.String(reposFlagName), ",")
	}
	return repos
}

func getAnalyzers(params common.GitHubRepositoryParams, ctx cli.Context) []spec.Analyzer {
	var authorName string
	if ctx.Has(common.CopyrightAuthorFlagName) {
		authorName = ctx.String(common.CopyrightAuthorFlagName)
	}

	return []spec.Analyzer{
		spec.NewDescriptionAnalyzer(),
		spec.NewOwnersAnalyzer(),
		spec.NewLicenseAnalyzer(params.CachingOAuthGitHubClient(), authorName),
		spec.NewHasPatentsAnalyzer(),
	}
}

func doCreateSpec(params common.GitHubRepositoryParams, repos []string, outputFile string, stdout io.Writer) error {
	client := params.CachingOAuthGitHubClient()
	var defs []repository.Definition
	if err := params.ProcessRepos(client, repos, func(repo *github.Repository, progress repository.Progress) error {
		fmt.Fprintf(stdout, "Generating definition for %s...", *repo.FullName)
		defer func() {
			fmt.Fprintln(stdout)
		}()
		info, err := repository.GetInfo(client, repo)
		if err != nil {
			fmt.Fprintf(stdout, "failed")
			return err
		}
		defs = append(defs, info.ToDefinition())
		fmt.Fprintf(stdout, "done")
		return nil
	}); err != nil {
		return err
	}
	sort.Sort(repository.DefinitionSlice(defs))

	bytes, err := yaml.Marshal(defs)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %v as YML", defs)
	}
	if err := ioutil.WriteFile(outputFile, bytes, 0644); err != nil {
		return errors.Wrapf(err, "failed to write file %s", outputFile)
	}

	fmt.Fprintf(stdout, "Wrote definitions for %s to %s\n", pluralizedRepositories(len(defs)), outputFile)
	return nil
}

type specMode int

const (
	verifyMode specMode = iota
	applyMode
)

func processSpec(params common.GitHubRepositoryParams, repos []string, specFile string, analyzers []spec.Analyzer, mode specMode, prompt bool, stdout io.Writer) error {
	bytes, err := ioutil.ReadFile(specFile)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s", specFile)
	}

	var defs []repository.Definition
	if err := yaml.Unmarshal(bytes, &defs); err != nil {
		return errors.Wrapf(err, "failed to unmarshal %s", string(bytes))
	}

	defsMap := make(map[string]repository.Definition, len(defs))
	missingReposSet := make(map[string]struct{}, len(defs)) // in definition file but not in GitHub
	for _, def := range defs {
		defsMap[def.FullName] = def
		missingReposSet[def.FullName] = struct{}{}
	}

	var unexpectedRepos []string               // not in definition file but in GitHub
	diffRepos := make(map[string]string)       // repos that exist but differ from definition
	var okRepos []string                       // repos that match specification
	var fixedRepos []string                    // repos successfully fixed
	failedToFixRepos := make(map[string]error) // repos not successfully fixed (value is error encountered)

	client := params.CachingOAuthGitHubClient()
	if err := params.ProcessRepos(client, repos, func(repo *github.Repository, progress repository.Progress) error {
		fmt.Fprintf(stdout, "Verifying repository %s against definition (%v)...", *repo.Name, progress)

		wantDef, ok := defsMap[*repo.FullName]
		if !ok {
			unexpectedRepos = append(unexpectedRepos, *repo.FullName)
			fmt.Fprintln(stdout, "no definition for repository")
			return nil
		}

		info, err := repository.GetInfo(client, repo)
		if err != nil {
			fmt.Fprintln(stdout, "failed to get repository info")
			return err
		}

		delete(missingReposSet, *repo.FullName)

		var diffs []string
		for _, analyzer := range analyzers {
			diff := analyzer.Diff(wantDef, info)
			if diff != "" {
				diffs = append(diffs, diff)
			}
		}

		if len(diffs) == 0 {
			okRepos = append(okRepos, *repo.FullName)
			fmt.Fprintln(stdout, "OK")
			return nil
		}

		diffRepos[*repo.FullName] = strings.Join(diffs, "\n")
		fmt.Fprintln(stdout, "differs from definition")

		if mode != applyMode {
			return nil
		}

		if prompt {
			ok, err := common.Prompt("Open PR for fix", stdout)
			if err != nil {
				return errors.Wrapf(err, "prompt failed")
			}
			if !ok {
				// if user is given prompt and provides non-"Yes" response, skip
				failedToFixRepos[*repo.FullName] = errors.Errorf("user skipped fix")
				return nil
			}
		}

		for _, analyzer := range analyzers {
			if !analyzer.CanFix() {
				continue
			}
			fmt.Fprintf(stdout, "Fixing %s for repository %s...", analyzer.Name(), *repo.FullName)

			if err := analyzer.Fix(wantDef, info, stdout); err != nil {
				failedToFixRepos[*repo.FullName] = err
				fmt.Fprintln(stdout, "failed")
				return nil
			}

			fmt.Fprintln(stdout, "OK")
		}

		fixedRepos = append(fixedRepos, *repo.FullName)
		return nil
	}); err != nil {
		return err
	}

	if mode == verifyMode {
		var errMsgParts []string
		if len(unexpectedRepos) > 0 {
			sort.Sort(repository.CaseInsensitiveStrings(unexpectedRepos))
			unexpectedParts := []string{fmt.Sprintf("%s without definitions:", pluralizedRepositories(len(unexpectedRepos)))}
			unexpectedParts = append(unexpectedParts, unexpectedRepos...)
			errMsgParts = append(errMsgParts, strings.Join(unexpectedParts, "\n\t"))
		}
		if len(missingReposSet) > 0 {
			missingRepos := make([]string, 0, len(missingReposSet))
			for k := range missingReposSet {
				missingRepos = append(missingRepos, k)
			}
			sort.Sort(repository.CaseInsensitiveStrings(missingRepos))
			missingParts := []string{fmt.Sprintf("%s missing:", pluralizedRepositories(len(missingRepos)))}
			missingParts = append(missingParts, missingRepos...)
			errMsgParts = append(errMsgParts, strings.Join(missingParts, "\n\t"))
		}
		if len(diffRepos) > 0 {
			errMsgParts = append(errMsgParts, fmt.Sprintf("%s differed from definition:", pluralizedRepositories(len(diffRepos))))

			keys := make([]string, 0, len(diffRepos))
			for k := range diffRepos {
				keys = append(keys, k)
			}
			sort.Sort(repository.CaseInsensitiveStrings(keys))

			for _, k := range keys {
				for _, v := range strings.Split(diffRepos[k], "\n") {
					errMsgParts = append(errMsgParts, "\t"+v)
				}
			}
		}
		if len(okRepos) > 0 {
			sort.Sort(repository.CaseInsensitiveStrings(okRepos))
			okParts := []string{fmt.Sprintf("%s OK", pluralizedRepositories(len(okRepos)))}
			okParts = append(okParts, okRepos...)
			fmt.Fprintln(stdout, strings.Join(okParts, "\n\t"))
		}
		if len(errMsgParts) > 0 {
			return errors.Errorf("%s", strings.Join(errMsgParts, "\n"))
		}
		return nil
	}

	if len(missingReposSet) > 0 {
		for k := range missingReposSet {
			failedToFixRepos[k] = errors.Errorf("repository not present (creating repos not implemented)")
		}
	}
	if len(okRepos) > 0 {
		sort.Sort(repository.CaseInsensitiveStrings(okRepos))
		okParts := []string{fmt.Sprintf("%s OK", pluralizedRepositories(len(okRepos)))}
		okParts = append(okParts, okRepos...)
		fmt.Fprintln(stdout, strings.Join(okParts, "\n\t"))
	}
	if len(fixedRepos) > 0 {
		fixedParts := []string{fmt.Sprintf("%s fixed", pluralizedRepositories(len(okRepos)))}
		fixedParts = append(fixedParts, fixedRepos...)
		fmt.Fprintln(stdout, strings.Join(fixedParts, "\n\t"))
	}
	if len(failedToFixRepos) > 0 {
		failedToFixKeys := make([]string, 0, len(failedToFixRepos))
		for k := range failedToFixRepos {
			failedToFixKeys = append(failedToFixKeys, k)
		}
		sort.Sort(repository.CaseInsensitiveStrings(failedToFixKeys))

		failedParts := []string{fmt.Sprintf("Failed to fix %s:", pluralizedRepositories(len(failedToFixKeys)))}
		for _, k := range failedToFixKeys {
			failedParts = append(failedParts, "%s: %s", k, failedToFixRepos[k].Error())
		}
		return errors.Errorf("%s", strings.Join(failedParts, "\n\t"))
	}
	return nil
}

func pluralizedRepositories(num int) string {
	str := fmt.Sprintf("%d", num)
	if num == 1 {
		return str + " repository"
	}
	return str + " repositories"
}
