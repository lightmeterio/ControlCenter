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
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
	"os"
)

func ReadFromTestReaderWithLogFormat(reader io.Reader, pub postfix.Publisher, year int, format string, clock timeutil.Clock) {
	builder, err := transform.Get(format, clock, year)
	errorutil.MustSucceed(err)

	s, err := filelogsource.New(reader, builder, &announcer.DummyImportAnnouncer{})
	errorutil.MustSucceed(err)

	r := logsource.NewReader(s, pub)
	err = r.Run()
	errorutil.MustSucceed(err)
}

func ReadFromTestReader(reader io.Reader, pub postfix.Publisher, year int, clock timeutil.Clock) {
	ReadFromTestReaderWithLogFormat(reader, pub, year, "default", clock)
}

func openFile(name string) *os.File {
	f, err := os.Open(name)
	errorutil.MustSucceed(err)

	return f
}

func ReadFromTestFileWithFormat(name string, pub postfix.Publisher, year int, format string, clock timeutil.Clock) {
	f := openFile(name)
	ReadFromTestReaderWithLogFormat(f, pub, year, format, clock)
}

func ReadFromTestFile(name string, pub postfix.Publisher, year int, clock timeutil.Clock) {
	ReadFromTestFileWithFormat(name, pub, year, "default", clock)
}
