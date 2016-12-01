package spec

import (
	"fmt"
	"io"
	"sort"

	"github.com/pkg/errors"

	"github.com/nmiyake/ghcli/repository"
)

type ownersAnalyzer struct{}

func NewOwnersAnalyzer() Analyzer {
	return &ownersAnalyzer{}
}

func (d *ownersAnalyzer) Name() string {
	return "owners"
}

func (d *ownersAnalyzer) Diff(def repository.Definition, info repository.Info) string {
	if len(def.Owners) == 0 || len(info.Owners) == 0 {
		return ""
	}
	missing := difference(def.Owners, info.Owners)
	if len(missing) == 0 {
		return ""
	}
	return joinDiff(
		d.Name(),
		fmt.Sprintf("required: %v", def.Owners),
		fmt.Sprintf("got:      %v", info.Owners),
		fmt.Sprintf("missing:  %v", missing),
	)
}

func (d *ownersAnalyzer) CanFix() bool {
	return false
}

// TODO(nmiyake): fix by adding owners
func (d *ownersAnalyzer) Fix(def repository.Definition, info repository.Info, stdout io.Writer) error {
	return errors.Errorf("not implemented")
}

// Returns the elements in want that are not in got. Returned slice is sorted using case-insensitive sort.
func difference(want, got []string) []string {
	var diff []string
	gotSet := toSet(got)
	for _, k := range want {
		if _, ok := gotSet[k]; !ok {
			diff = append(diff, k)
		}
	}
	sort.Sort(repository.CaseInsensitiveStrings(diff))
	return diff
}

func toSet(input []string) map[string]struct{} {
	m := make(map[string]struct{}, len(input))
	for i := range input {
		m[input[i]] = struct{}{}
	}
	return m
}
