// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package settingsutil

import (
	"context"

	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func Set[T any](ctx context.Context, writer *metadata.AsyncWriter, settings T, key string) error {
	if err := writer.StoreJsonSync(ctx, key, settings); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func Get[T any](ctx context.Context, reader metadata.Reader, key string) (*T, error) {
	var settings T

	if err := reader.RetrieveJson(ctx, key, &settings); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &settings, nil
}
