// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

/**
This is a simple program that sends a report to the network inteligence server, mostly to check if it's still working.

```
./example -postfix_ip 127.0.0.2 -public_url http://example.com/lightmeter -server_url http://localhost:8080/reports
```

*/

package main

import (
	"context"
	"flag"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/intel"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func main() {
	serverURL := flag.String("server_url", "", "URL for the endppoint that receives the reports")
	postfixIP := flag.String("postfix_ip", "", "Postfix IP address")
	publicURL := flag.String("public_url", "", "Public Lightmeter url")

	flag.Parse()

	log.Printf("%v, %v, %v", *serverURL, *postfixIP, *publicURL)

	dir, err := ioutil.TempDir("", "")
	errorutil.MustSucceed(err)

	defer os.RemoveAll(dir)

	conn, err := dbconn.Open(path.Join(dir, "master.db"), 5)
	errorutil.MustSucceed(err)

	defer conn.Close()

	err = migrator.Run(conn.RwConn.DB, "master")
	errorutil.MustSucceed(err)

	m, err := metadata.NewHandler(conn)
	errorutil.MustSucceed(err)

	err = m.Writer.StoreJson(context.Background(), globalsettings.SettingKey, globalsettings.Settings{
		LocalIP:     net.ParseIP(*postfixIP),
		APPLanguage: "en",
		PublicURL:   *publicURL,
	})

	errorutil.MustSucceed(err)

	auth := &auth.FakeRegistrar{Email: "user@example.com"}

	dispatcher := intel.Dispatcher{
		ReportDestinationURL: *serverURL,
		SettingsReader:       m.Reader,
		VersionBuilder:       intel.DefaultVersionBuilder,
		InstanceID:           "8946c49f-22ee-4577-bcbc-121ac8c715c9",
		Auth:                 auth,
		SchedFileReader:      intel.DefaultSchedFileReader,
	}

	err = dispatcher.Dispatch(collector.Report{
		Interval: timeutil.TimeInterval{From: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`), To: timeutil.MustParseTime(`2000-01-01 10:00:00 +0000`)},
		Content: []collector.ReportEntry{
			{Time: timeutil.MustParseTime(`2000-01-01 01:00:00 +0000`), ID: "some_id", Payload: "some_payload"},
		},
	})

	errorutil.MustSucceed(err)
}
