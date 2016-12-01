package spec

import (
	"fmt"
	"io"
	"strings"

	"github.com/nmiyake/ghcli/repository"
)

type Analyzer interface {
	Name() string
	Diff(def repository.Definition, info repository.Info) string
	CanFix() bool
	Fix(def repository.Definition, info repository.Info, stdout io.Writer) error
}

func stringDiff(name, want, got string) string {
	if want == got {
		return ""
	}
	return joinDiff(
		name,
		fmt.Sprintf("want: %s", want),
		fmt.Sprintf("got:  %s", got),
	)
}

func joinDiff(name string, content ...string) string {
	parts := []string{name + ":"}
	for _, curr := range content {
		parts = append(parts, "\t"+curr)
	}
	return strings.Join(parts, "\n")
}

func orEmpty(in *string) string {
	if in == nil {
		return ""
	}
	return *in
}
