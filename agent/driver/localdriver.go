// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package driver

import (
	"context"
	"errors"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"os/exec"
)

type LocalDriver struct {
}

var ErrInvalidCommand = errors.New(`Invalid Command`)

func (d *LocalDriver) ExecuteCommand(ctx context.Context, command []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(command) < 1 {
		return ErrInvalidCommand
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)

	if stdin != nil {
		cmd.Stdin = stdin
	}

	if stdout != nil {
		cmd.Stdout = stdout
	}

	if stderr != nil {
		cmd.Stderr = stderr
	}

	if err := cmd.Run(); err != nil {
		errorutil.Wrap(err)
	}

	return nil
}
