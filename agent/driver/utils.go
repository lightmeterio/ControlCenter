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

func ReadFileContent(ctx context.Context, driver Driver, filepath string, dst io.Writer) error {
	// then get the content
	if err := driver.ExecuteCommand(context.Background(), []string{"cat", filepath}, nil, dst, io.Discard); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func WriteFileContent(ctx context.Context, driver Driver, filepath string, src io.Reader) error {
	catCommand := fmt.Sprintf(`cat > %s`, filepath)

	if err := driver.ExecuteCommand(context.Background(), []string{`sh`, `-c`, catCommand}, src, io.Discard, io.Discard); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
