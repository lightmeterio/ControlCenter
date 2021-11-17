// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
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
	LogPatterns               []string
	Address                   string
	LogLevel                  zerolog.Level
	Timezone                  *time.Location
	LogYear                   int
	Socket                    string
	LogFormat                 string

	EmailToChange          string
	PasswordToReset        string
	ChangeUserInfoNewEmail string
	ChangeUserInfoNewName  string
}

func Parse(cmdlineArgs []string, lookupenv func(string) (string, bool)) (Config, error) {
	return ParseWithErrorHandling(cmdlineArgs, lookupenv, flag.ExitOnError)
}

func ParseWithErrorHandling(cmdlineArgs []string, lookupenv func(string) (string, bool), errorHandling flag.ErrorHandling) (Config, error) {
	var conf Config
	conf.Timezone = time.UTC

	// new flagset to be able to call ParseFlags any number of times
	fs := flag.NewFlagSet("our_flag_set", errorHandling)

	fs.BoolVar(&conf.ShouldWatchFromStdin, "stdin", false, "Read log lines from stdin")

	fs.StringVar(&conf.WorkspaceDirectory, "workspace",
		lookupEnvOrString("LIGHTMETER_WORKSPACE", "/var/lib/lightmeter_workspace", lookupenv),
		"Path to the directory to store all working data")

	fs.BoolVar(&conf.ImportOnly, "importonly", false,
		"Only import existing logs, exiting immediately, without running the full application.")

	b, err := lookupEnvOrBool("LIGHTMETER_LOGS_USE_RSYNC", false, lookupenv)
	if err != nil {
		return conf, err
	}

	fs.BoolVar(&conf.RsyncedDir, "logs_use_rsync", b, "Log directory is updated by rsync")

	fs.BoolVar(&conf.MigrateDownToOnly, "migrate_down_to_only", false,
		"Only migrates down")
	fs.StringVar(&conf.MigrateDownToDatabaseName, "migrate_down_to_database", "", "Database name only for migration")

	fs.IntVar(&conf.MigrateDownToVersion, "migrate_down_to_version", -1, "Specify the new migration version")

	logYear, err := lookupEnvOrInt("LIGHTMETER_LOGS_STARTING_YEAR", 0, lookupenv)
	if err != nil {
		return conf, err
	}

	fs.IntVar(&conf.LogYear, "log_starting_year", int(logYear),
		"Value to be used as initial year when it cannot be obtained from the Postfix logs. Defaults to the current year. Requires -stdin or -socket")

	fs.BoolVar(&conf.ShowVersion, "version", false, "Show Version Information")

	fs.StringVar(&conf.DirToWatch, "watch_dir",
		lookupEnvOrString("LIGHTMETER_WATCH_DIR", "", lookupenv),
		"Path to the directory where postfix stores its log files, to be watched")

	fs.StringVar(&conf.Address, "listen",
		lookupEnvOrString("LIGHTMETER_LISTEN", ":8080", lookupenv),
		"Network Address to listen to")

	var stringLogLevel = "INFO"

	fs.StringVar(&stringLogLevel, "log_level",
		lookupEnvOrString("LIGHTMETER_LOG_LEVEL", "INFO", lookupenv),
		"Log level (DEBUG, INFO, WARN, or ERROR. Default: INFO)")

	fs.StringVar(&conf.EmailToChange, "email_reset", "", "Change user info (email, name or password; depends on -workspace)")
	fs.StringVar(&conf.PasswordToReset, "password", "", "Password to reset (requires -email_reset)")

	fs.StringVar(&conf.ChangeUserInfoNewEmail, "new_email", "", "Update user email (depends on -email_reset)")
	fs.StringVar(&conf.ChangeUserInfoNewName, "new_user_name", "", "Update user name (depends on -email_reset)")

	fs.StringVar(&conf.Socket, "logs_socket",
		lookupEnvOrString("LIGHTMETER_LOGS_SOCKET", "", lookupenv),
		"Receive logs via a Socket. E.g. unix=/tmp/lightemter.sock or tcp=localhost:9999")

	fs.StringVar(&conf.LogFormat, "log_format",
		lookupEnvOrString("LIGHTMETER_LOG_FORMAT", "default", lookupenv),
		"Expected log format from external sources (like logstash, etc.)")

	var unparsedLogPatterns string

	fs.StringVar(&unparsedLogPatterns, "log_file_patterns", lookupEnvOrString("LIGHTMETER_LOG_FILE_PATTERNS", "", lookupenv),
		`An optional colon separated list of the base filenames for the Postfix log files. Example: "mail.log:mail.err:mail.log" or "maillog"`)

	fs.Usage = func() {
		version.PrintVersion()
		fmt.Fprintf(os.Stdout, "\n Example call: \n")
		fmt.Fprintf(os.Stdout, "\n %s -workspace ~/lightmeter_workspace -watch_dir /var/log \n", os.Args[0])
		fmt.Fprintf(os.Stdout, "\n Flag set: \n\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(cmdlineArgs); err != nil {
		return Config{}, errorutil.Wrap(err)
	}

	conf.LogLevel, err = zerolog.ParseLevel(strings.ToLower(stringLogLevel))
	if err != nil {
		return conf, err
	}

	conf.LogPatterns = func() []string {
		p := strings.Split(unparsedLogPatterns, ":")

		if len(p) == 1 && p[0] == "" {
			return []string{}
		}

		return p
	}()

	return conf, nil
}

func lookupEnvOrString(key string, defaultVal string, loopkupenv func(string) (string, bool)) string {
	if val, ok := loopkupenv(key); ok {
		return val
	}

	return defaultVal
}

func lookupEnvOrBool(key string, defaultVal bool, loopkupenv func(string) (string, bool)) (bool, error) {
	if val, ok := loopkupenv(key); ok {
		v, err := strconv.ParseBool(val)
		if err != nil {
			return v, fmt.Errorf("Boolean env var %v boolean value could not be parsed: %w", key, err)
		}

		return v, nil
	}

	return defaultVal, nil
}

func lookupEnvOrInt(key string, defaultVal int64, loopkupenv func(string) (string, bool)) (int64, error) {
	if val, ok := loopkupenv(key); ok {
		v, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return v, fmt.Errorf("Integer env var %v integer value could not be parsed: %w", key, err)
		}

		return v, nil
	}

	return defaultVal, nil
}
