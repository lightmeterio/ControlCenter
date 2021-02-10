// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package recommendation

import "sync"

var defaultURLContainer URLContainer
var once sync.Once
var links = make([]Link, 0)

func GetDefaultURLContainer() URLContainer {
	once.Do(func() {
		defaultURLContainer = NewURLContainer()
		defaultURLContainer.SetForEach(links)
	})

	return defaultURLContainer
}
