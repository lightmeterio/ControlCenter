// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gitlab.com/lightmeter/controlcenter/util/envvarsutil"
	"gitlab.com/lightmeter/controlcenter/version"
)

// nolint: maligned
type Config struct {
	ShouldWatchFromStdin      bool
	WorkspaceDirectory        string
	ImportOnly                bool
	RsyncedDir                bool
	MigrateDownToOnly         bool
	MigrateDownToVersion      int
	MigrateDownToDatabaseName string
	ShowVersion               bool
	DirToWatch                string
	Address                   string
	Verbose                   bool
	EmailToPasswdReset        string
	PasswordToReset           string
	Timezone                  *time.Location
	LogYear                   int
	Socket                    string
	LogFormat                 string
}

func ParseConfig(cmdlineArgs []string, lookupenv func(string) (string, bool)) (Config, error) {
	var conf Config
	conf.Timezone = time.UTC

	// new flagset to be able to call ParseFlags any number of times
	fs := flag.NewFlagSet("our_flag_set", flag.ExitOnError)
	fs.BoolVar(&conf.ShouldWatchFromStdin, "stdin", false, "Read log lines from stdin")
	fs.StringVar(&conf.WorkspaceDirectory, "workspace",
		envvarsutil.LookupEnvOrString("LIGHTMETER_WORKSPACE", "/var/lib/lightmeter_workspace", lookupenv),
		"Path to the directory to store all working data")
	fs.BoolVar(&conf.ImportOnly, "ImportOnly", false,
		"Only import existing logs, exiting immediately, without running the full application.")
	fs.BoolVar(&conf.RsyncedDir, "logs_use_rsync",
		envvarsutil.LookupEnvOrBool("LIGHTMETER_LOGS_USE_RSYNC", false, lookupenv),
		"Log directory is updated by rsync")
	fs.BoolVar(&conf.MigrateDownToOnly, "migrate_down_to_only", false,
		"Only migrates down")
	fs.StringVar(&conf.MigrateDownToDatabaseName, "migrate_down_to_database", "", "Database name only for migration")
	fs.IntVar(&conf.MigrateDownToVersion, "migrate_down_to_version", -1, "Specify the new migration version")
	fs.IntVar(&conf.LogYear, "log_starting_year", 0, "Value to be used as initial year when it cannot be obtained from the Postfix logs. Defaults to the current year. Requires -stdin.")
	fs.BoolVar(&conf.ShowVersion, "version", false, "Show Version Information")
	fs.StringVar(&conf.DirToWatch, "watch_dir",
		envvarsutil.LookupEnvOrString("LIGHTMETER_WATCH_DIR", "", lookupenv),
		"Path to the directory where postfix stores its log files, to be watched")
	fs.StringVar(&conf.Address, "listen",
		envvarsutil.LookupEnvOrString("LIGHTMETER_LISTEN", ":8080", lookupenv),
		"Network Address to listen to")
	fs.BoolVar(&conf.Verbose, "Verbose",
		envvarsutil.LookupEnvOrBool("LIGHTMETER_VERBOSE", false, lookupenv),
		"Be Verbose")
	fs.StringVar(&conf.EmailToPasswdReset, "email_reset", "", "Reset password for user (implies -password and depends on -workspace)")
	fs.StringVar(&conf.PasswordToReset, "password", "", "Password to reset (requires -email_reset)")
	fs.StringVar(&conf.Socket, "logs_Socket",
		envvarsutil.LookupEnvOrString("LIGHTMETER_LOGS_SOCKET", "", lookupenv),
		"Receive logs via a Socket. E.g. unix=/tmp/lightemter.sock or tcp=localhost:9999")
	fs.StringVar(&conf.LogFormat, "log_format",
		envvarsutil.LookupEnvOrString("LIGHTMETER_LOG_FORMAT", "default", lookupenv),
		"Expected log format from external sources (like logstash, etc.)")

	fs.Usage = func() {
		version.PrintVersion()
		fmt.Fprintf(os.Stdout, "\n Example call: \n")
		fmt.Fprintf(os.Stdout, "\n %s -workspace ~/lightmeter_workspace -watch_dir /var/log \n", os.Args[0])
		fmt.Fprintf(os.Stdout, "\n Flag set: \n\n")
		fs.PrintDefaults()
	}

	_ = fs.Parse(cmdlineArgs) // ErrHelp should never happen since our -help/-h flag is defined

	return conf, nil
}
