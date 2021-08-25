package driver

// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

import (
	"context"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
)

var (
	Stdout io.Writer = nil
	Stderr io.Writer = nil
)

func ReadFileContent(ctx context.Context, driver Driver, filepath string, dst io.Writer) error {
	// then get the content
	if err := driver.ExecuteCommand(context.Background(), []string{"cat", filepath}, nil, dst, Stderr); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func WriteFileContent(ctx context.Context, driver Driver, filepath string, src io.Reader) error {
	catCommand := fmt.Sprintf(`cat > %s`, filepath)

	if err := driver.ExecuteCommand(context.Background(), []string{`sh`, `-c`, catCommand}, src, Stdout, Stderr); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
