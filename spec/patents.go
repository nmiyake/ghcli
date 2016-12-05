// Copyright 2016 Nick Miyake. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package spec

import (
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/nmiyake/ghcli/repository"
)

type hasPatentsAnalyzer struct{}

func NewHasPatentsAnalyzer() Analyzer {
	return &hasPatentsAnalyzer{}
}

func (d *hasPatentsAnalyzer) Name() string {
	return "patents"
}

func (d *hasPatentsAnalyzer) Diff(def repository.Definition, info repository.Info) string {
	if def.HasPatents == info.HasPatents {
		return ""
	}
	return joinDiff(
		"has patents",
		fmt.Sprintf("want: %v", def.HasPatents),
		fmt.Sprintf("got:  %v", info.HasPatents),
	)
}

func (d *hasPatentsAnalyzer) CanFix() bool {
	return false
}

// TODO(nmiyake): fix by adding PATENTS file
func (d *hasPatentsAnalyzer) Fix(def repository.Definition, info repository.Info, stdout io.Writer) error {
	return errors.Errorf("not implemented")
}
