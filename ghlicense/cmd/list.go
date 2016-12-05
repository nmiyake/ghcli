// Copyright 2016 Nick Miyake. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/palantir/pkg/cli"
	"github.com/palantir/pkg/cli/flag"
	"github.com/pkg/errors"

	"github.com/nmiyake/ghcli/common"
	"github.com/nmiyake/ghcli/license"
)

const (
	headerFlagName  = "header"
	aliasesFlagName = "aliases"
)

var (
	headerFlag = flag.BoolFlag{
		Name:  headerFlagName,
		Usage: "display header",
		Value: true,
	}
	aliasesFlag = flag.BoolFlag{
		Name:  aliasesFlagName,
		Usage: "display aliases",
		Value: true,
	}
)

func List() cli.Command {
	gitHubTokenFlag := common.GitHubTokenFlag
	gitHubTokenFlag.Required = false
	return cli.Command{
		Name:  "list",
		Usage: "list all available licenses",
		Flags: []flag.Flag{
			gitHubTokenFlag,
			headerFlag,
			aliasesFlag,
		},
		Action: func(ctx cli.Context) error {
			return doList(ctx.String(common.GitHubTokenFlagName), ctx.Bool(headerFlagName), ctx.Bool(aliasesFlagName), ctx.App.Stdout)
		},
	}
}

func doList(gitHubToken string, header, aliases bool, stdout io.Writer) error {
	client := common.CachingOAuthGitHubClient(gitHubToken, "")

	licenses, _, err := client.Licenses.List()
	if err != nil {
		return errors.Wrapf(err, "failed to list licenses")
	}

	licenseIDs := make([]string, len(licenses))
	for i := range licenseIDs {
		licenseIDs[i] = *licenses[i].SPDXID
	}
	sort.Strings(licenseIDs)

	var output string
	if !header && !aliases {
		output = strings.Join(append(licenseIDs, ""), "\n")
	} else {
		buf := &bytes.Buffer{}
		tw := tabwriter.NewWriter(buf, 0, 0, 4, ' ', uint(0))

		if header {
			titles := "ID\t"
			if aliases {
				titles += "ALIASES\t"
			}
			fmt.Fprintln(tw, titles)
		}

		for _, currID := range licenseIDs {
			currRow := currID + "\t"
			if aliases {
				if currAliases := license.Aliases(currID); currAliases != nil {
					currRow += fmt.Sprintf("%v\t", strings.Join(currAliases, ", "))
				}
			}
			fmt.Fprintln(tw, currRow)
		}
		_ = tw.Flush()
		output = buf.String()
	}
	fmt.Fprint(stdout, output)
	return nil
}
