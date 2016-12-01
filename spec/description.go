package spec

import (
	"io"

	"github.com/pkg/errors"

	"github.com/nmiyake/ghcli/repository"
)

type descriptionAnalyzer struct{}

func NewDescriptionAnalyzer() Analyzer {
	return &descriptionAnalyzer{}
}

func (d *descriptionAnalyzer) Name() string {
	return "description"
}

func (d *descriptionAnalyzer) Diff(def repository.Definition, info repository.Info) string {
	return stringDiff(d.Name(), def.Description, orEmpty(info.Description))
}

func (d *descriptionAnalyzer) CanFix() bool {
	return false
}

// TODO(nmiyake): fix by using API to change description
func (d *descriptionAnalyzer) Fix(def repository.Definition, info repository.Info, stdout io.Writer) error {
	return errors.Errorf("not implemented")
}
