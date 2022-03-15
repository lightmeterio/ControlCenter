// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/envutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/version"
)

type dirs []string

func (d *dirs) String() string {
	return strings.Join(*d, ", ")
}

func (d *dirs) Set(s string) error {
	*d = append(*d, s)
	return nil
}

// nolint: maligned
type Config struct {
	ShouldWatchFromStdin bool
	WorkspaceDirectory   string
	ImportOnly           bool
	RsyncedDir           bool
	ShowVersion          bool
	DirsToWatch          []string
	LogPatterns          []string
	Address              string
	LogLevel             zerolog.Level
	Timezone             *time.Location
	LogYear              int
	Socket               string
	LogFormat            string

	EmailToChange          string
	PasswordToReset        string
	ChangeUserInfoNewEmail string
	ChangeUserInfoNewName  string

	// set it when control center is **NOT** behind a reverse proxy,
	// being accessed directly, on plain HTTP (as on 2.0)
	IKnowWhatIAmDoingNotUsingAReverseProxy bool

	DefaultSettings        metadata.DefaultValues
	RegisteredUserName     string
	RegisteredUserEmail    string
	RegisteredUserPassword string

	GenerateDovecotConfig bool
	DovecotConfigIsOld    bool
}

func Parse(cmdlineArgs []string, lookupenv func(string) (string, bool)) (Config, error) {
	return ParseWithErrorHandling(cmdlineArgs, lookupenv, flag.ExitOnError)
}

func ParseWithErrorHandling(cmdlineArgs []string, lookupenv func(string) (string, bool), errorHandling flag.ErrorHandling) (Config, error) {
	conf := Config{DefaultSettings: metadata.DefaultValues{}}
	conf.Timezone = time.UTC

	// new flagset to be able to call ParseFlags any number of times
	fs := flag.NewFlagSet("our_flag_set", errorHandling)

	fs.BoolVar(&conf.ShouldWatchFromStdin, "stdin", false, "Read log lines from stdin")

	fs.StringVar(&conf.WorkspaceDirectory, "workspace",
		envutil.LookupEnvOrString("LIGHTMETER_WORKSPACE", "/var/lib/lightmeter_workspace", lookupenv),
		"Path to the directory to store all working data")

	fs.BoolVar(&conf.ImportOnly, "importonly", false,
		"Only import existing logs, exiting immediately, without running the full application.")

	b, err := envutil.LookupEnvOrBool("LIGHTMETER_LOGS_USE_RSYNC", false, lookupenv)
	if err != nil {
		return conf, err
	}

	fs.BoolVar(&conf.RsyncedDir, "logs_use_rsync", b, "Log directory is updated by rsync")

	logYear, err := envutil.LookupEnvOrInt("LIGHTMETER_LOGS_STARTING_YEAR", 0, lookupenv)
	if err != nil {
		return conf, err
	}

	fs.IntVar(&conf.LogYear, "log_starting_year", int(logYear),
		"Value to be used as initial year when it cannot be obtained from the Postfix logs. Defaults to the current year. Requires -stdin or -socket")

	fs.BoolVar(&conf.ShowVersion, "version", false, "Show Version Information")

	dirsToWatchFromEnvironment := strings.Split(envutil.LookupEnvOrString("LIGHTMETER_WATCH_DIR", "", lookupenv), ":")

	var dirsToWatch dirs

	fs.Var(&dirsToWatch, "watch_dir", "Path to the directories where postfix stores its log files, to be watched")

	fs.StringVar(&conf.Address, "listen",
		envutil.LookupEnvOrString("LIGHTMETER_LISTEN", ":8080", lookupenv),
		"Network Address to listen to")

	var stringLogLevel = "INFO"

	fs.StringVar(&stringLogLevel, "log_level",
		envutil.LookupEnvOrString("LIGHTMETER_LOG_LEVEL", "INFO", lookupenv),
		"Log level (DEBUG, INFO, WARN, or ERROR. Default: INFO)")

	fs.StringVar(&conf.EmailToChange, "email_reset", "", "Change user info (email, name or password; depends on -workspace)")
	fs.StringVar(&conf.PasswordToReset, "password", "", "Password to reset (requires -email_reset)")

	fs.StringVar(&conf.ChangeUserInfoNewEmail, "new_email", "", "Update user email (depends on -email_reset)")
	fs.StringVar(&conf.ChangeUserInfoNewName, "new_user_name", "", "Update user name (depends on -email_reset)")

	fs.StringVar(&conf.Socket, "logs_socket",
		envutil.LookupEnvOrString("LIGHTMETER_LOGS_SOCKET", "", lookupenv),
		"Receive logs via a Socket. E.g. unix=/tmp/lightemter.sock or tcp=localhost:9999")

	fs.StringVar(&conf.LogFormat, "log_format",
		envutil.LookupEnvOrString("LIGHTMETER_LOG_FORMAT", "default", lookupenv),
		"Expected log format from external sources (like logstash, etc.)")

	var unparsedDefaultSettings string

	fs.StringVar(&unparsedDefaultSettings, "default_settings", envutil.LookupEnvOrString("LIGHTMETER_DEFAULT_SETTINGS", `{}`, lookupenv), "JSON string for default settings")

	var unparsedLogPatterns string

	fs.StringVar(&unparsedLogPatterns, "log_file_patterns", envutil.LookupEnvOrString("LIGHTMETER_LOG_FILE_PATTERNS", "", lookupenv),
		`An optional colon separated list of the base filenames for the Postfix log files. Example: "mail.log:mail.err:mail.log" or "maillog"`)

	proxyConf, err := envutil.LookupEnvOrBool("LIGHTMETER_I_KNOW_WHAT_I_AM_DOING_NOT_USING_A_REVERSE_PROXY", false, lookupenv)
	if err != nil {
		return conf, err
	}

	fs.BoolVar(&conf.IKnowWhatIAmDoingNotUsingAReverseProxy, "i_know_what_am_doing_not_using_a_reverse_proxy",
		proxyConf, "Used when you are accessing the application without a reverse proxy (e.g. apache2, nginx or traefik), "+
			"which is unsupported by us at the moment and might lead to security issues")

	fs.StringVar(&conf.RegisteredUserEmail, "registered_user_email", envutil.LookupEnvOrString("LIGHTMETER_REGISTERED_USER_EMAIL", "", lookupenv), "Experimental: static user e-mail")
	fs.StringVar(&conf.RegisteredUserName, "registered_user_name", envutil.LookupEnvOrString("LIGHTMETER_REGISTERED_USER_NAME", "", lookupenv), "Experimental: static user name")
	fs.StringVar(&conf.RegisteredUserPassword, "registered_user_password", envutil.LookupEnvOrString("LIGHTMETER_REGISTERED_USER_PASSWORD", "", lookupenv), "Experimental: static user password")

	fs.BoolVar(&conf.GenerateDovecotConfig, "dovecot_conf_gen", false, "Generate Dovecot Configuration")
	fs.BoolVar(&conf.DovecotConfigIsOld, "dovecot_conf_is_old", false, "Requires -dovecot_conf_gen. Use if if you're using a Dovecot older than 2.3.1")

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

	conf.DirsToWatch = buildDirsToWatch(dirsToWatch, dirsToWatchFromEnvironment)

	conf.LogLevel, err = zerolog.ParseLevel(strings.ToLower(stringLogLevel))
	if err != nil {
		return conf, err
	}

	d := json.NewDecoder(strings.NewReader(unparsedDefaultSettings))
	d.UseNumber()

	if err := d.Decode(&conf.DefaultSettings); err != nil {
		return Config{}, errorutil.Wrap(err)
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

func buildDirsToWatch(dirs dirs, dirsFromEnv []string) []string {
	if len(dirs) > 0 && len(dirs[0]) > 0 {
		return []string(dirs)
	}

	if len(dirsFromEnv) > 0 && len(dirsFromEnv[0]) > 0 {
		return dirsFromEnv
	}

	return nil
}
