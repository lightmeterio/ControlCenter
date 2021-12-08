// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package runner

import (
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

// TODO: handle errors!!!
func NewDependantPairCancellableRunner(dependency CancellableRunner, dependant CancellableRunner) CancellableRunner {
	return NewCancellableRunner(func(done DoneChan, cancel CancelChan) {
		dependencyDone, dependencyCancel := Run(dependency)
		dependantDone, dependantCancel := Run(dependant)

		go func() {
			<-cancel
			dependencyCancel()
			errorutil.MustSucceed(dependencyDone())
			dependantCancel()
		}()

		go func() {
			errorutil.MustSucceed(dependantDone())
			done <- nil
		}()
	})
}
