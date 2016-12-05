// Copyright 2016 Nick Miyake. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package common

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/palantir/pkg/cli/flag"
	"github.com/pkg/errors"
)

const (
	PromptFlagName = "prompt"
)

var (
	PromptFlag = flag.BoolFlag{
		Name:  PromptFlagName,
		Usage: "prompt before applying fixing repository (interactive)",
		Value: true,
	}
)

// Prompt displays the specified prompt and waits for input on os.Stdin. If the provided input represents a response of
// "Yes", the function returns true, otherwise, returns false.
func Prompt(prompt string, stdout io.Writer) (bool, error) {
	fmt.Fprintf(stdout, "%s (y/n): ", prompt)
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return false, errors.Wrapf(err, "ReadString failed")
	}
	return parseYesNo(text), nil
}

func parseYesNo(input string) bool {
	lower := strings.TrimSpace(strings.ToLower(input))
	return lower == "y" || lower == "yes"
}
