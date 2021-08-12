// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postfixutil

import (
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"os"
)

func ReadFromTestReader(reader io.Reader, pub postfix.Publisher, year int) {
	builder, err := transform.Get("default", year)
	errorutil.MustSucceed(err)

	s, err := filelogsource.New(reader, builder, &announcer.DummyImportAnnouncer{})
	errorutil.MustSucceed(err)

	r := logsource.NewReader(s, pub)
	err = r.Run()
	errorutil.MustSucceed(err)
}

func openFile(name string) *os.File {
	f, err := os.Open(name)
	errorutil.MustSucceed(err)

	return f
}

func ReadFromTestFile(name string, pub postfix.Publisher, year int) {
	f := openFile(name)
	ReadFromTestReader(f, pub, year)
}
