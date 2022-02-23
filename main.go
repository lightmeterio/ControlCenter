// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/config"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/dirlogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/dirwatcher"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	"gitlab.com/lightmeter/controlcenter/logeater/socketsource"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/server"
	"gitlab.com/lightmeter/controlcenter/subcommand"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"gitlab.com/lightmeter/controlcenter/version"
	"gitlab.com/lightmeter/controlcenter/workspace"
)

func changeUserInfo(conf config.Config) {
	if !(len(conf.ChangeUserInfoNewEmail) > 0 || len(conf.ChangeUserInfoNewName) > 0 || len(conf.PasswordToReset) > 0) {
		errorutil.Dief(nil, "No new user info to be changed")
	}

	subcommand.PerformUserInfoChange(
		conf.WorkspaceDirectory, conf.EmailToChange,
		conf.ChangeUserInfoNewEmail, conf.ChangeUserInfoNewName,
		conf.PasswordToReset,
	)
}

func main() {
	conf, err := config.Parse(os.Args[1:], os.LookupEnv)
	if err != nil {
		errorutil.Dief(errorutil.Wrap(err), "Could not parse command-line arguments or environment variables")
	}

	if conf.GenerateDovecotConfig {
		setupDovecotConfig(conf.DovecotConfigIsOld)
		return
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().Str("service", "controlcenter").Caller().Logger()

	zerolog.SetGlobalLevel(conf.LogLevel)

	if conf.ShowVersion {
		version.PrintVersion()
		return
	}

	liabilityDisclaimer := `This program comes with ABSOLUTELY NO WARRANTY. This is free software, and you are welcome to redistribute it under certain conditions; see here for details: https://lightmeter.io/lmcc-license.`

	log.Info().Msg(liabilityDisclaimer)

	lmsqlite3.Initialize(lmsqlite3.Options{})

	if len(conf.EmailToChange) > 0 {
		changeUserInfo(conf)
		return
	}

	ws, logReader, err := buildWorkspaceAndLogReader(conf)
	if err != nil {
		errorutil.Dief(errorutil.Wrap(err), "Error creating / opening workspace directory for storing application files: %s. Try specifying a different directory (using -workspace), or check you have permission to write to the specified location.", conf.WorkspaceDirectory)
	}

	done, cancel := runner.Run(ws)

	// only import logs and exit when they end. Does not start web server.
	// It's useful for benchmarking importing logs.
	if conf.ImportOnly {
		err := logReader.Run()

		if err != nil {
			errorutil.Dief(err, "Error reading logs")
		}

		cancel()

		err = done()

		errorutil.MustSucceed(err)

		log.Info().Msg("Importing has finished. Bye!")

		return
	}

	// from here on, workspace is never cancellable!

	go func() {
		err := done()
		errorutil.Dief(err, "Error: Workspace execution has ended, which should never happen here!")
	}()

	go func() {
		err := logReader.Run()
		if err != nil {
			errorutil.Dief(err, "Error reading logs")
		}
	}()

	httpServer := server.HttpServer{
		Workspace:            ws,
		WorkspaceDirectory:   conf.WorkspaceDirectory,
		Timezone:             conf.Timezone,
		Address:              conf.Address,
		IsBehindReverseProxy: !conf.IKnowWhatIAmDoingNotUsingAReverseProxy,
	}

	errorutil.MustSucceed(httpServer.Start(), "server died")
}

func buildAuthOptions(conf config.Config) auth.Options {
	if len(conf.RegisteredUserEmail) == 0 || len(conf.RegisteredUserName) == 0 || len(conf.RegisteredUserPassword) == 0 {
		return auth.Options{AllowMultipleUsers: false, PlainAuthOptions: nil}
	}

	log.Info().Msgf("Using user information from environment/command-line. This is VERY experimental: %v -> %v",
		conf.RegisteredUserEmail, conf.RegisteredUserName)

	return auth.Options{
		AllowMultipleUsers: false,
		PlainAuthOptions: &auth.PlainAuthOptions{
			Email:    conf.RegisteredUserEmail,
			Name:     conf.RegisteredUserName,
			Password: conf.RegisteredUserPassword,
		},
	}
}

func buildWorkspaceAndLogReader(conf config.Config) (*workspace.Workspace, logsource.Reader, error) {
	options := &workspace.Options{
		IsUsingRsyncedLogs: conf.RsyncedDir,
		DefaultSettings:    conf.DefaultSettings,
		AuthOptions:        buildAuthOptions(conf),
	}

	ws, err := workspace.NewWorkspace(conf.WorkspaceDirectory, options)
	if err != nil {
		return nil, logsource.Reader{}, errorutil.Wrap(err)
	}

	logSource, err := buildLogSource(ws, conf)
	if err != nil {
		return nil, logsource.Reader{}, errorutil.Wrap(err)
	}

	logReader := logsource.NewReader(logSource, ws.NewPublisher())

	return ws, logReader, nil
}

func buildLogSource(ws *workspace.Workspace, conf config.Config) (logsource.Source, error) {
	announcer, err := ws.ImportAnnouncer()
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	patterns := func(patterns []string) dirwatcher.LogPatterns {
		if len(patterns) == 0 {
			return dirwatcher.DefaultLogPatterns
		}

		return dirwatcher.BuildLogPatterns(patterns)
	}(conf.LogPatterns)

	if len(conf.DirToWatch) > 0 {
		sum, err := ws.MostRecentLogTimeAndSum()
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		s, err := dirlogsource.New(conf.DirToWatch, sum, announcer, !conf.ImportOnly, conf.RsyncedDir, conf.LogFormat, patterns, &timeutil.RealClock{})
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		return s, nil
	}

	builder, err := transform.Get(conf.LogFormat, conf.LogYear)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	if conf.ShouldWatchFromStdin {
		s, err := filelogsource.New(os.Stdin, builder, announcer)
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		return s, nil
	}

	if len(conf.Socket) > 0 {
		s, err := socketsource.New(conf.Socket, builder, announcer)
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		return s, nil
	}

	errorutil.Dief(nil, "No logs sources specified or import flag provided! Use -help to more info.")

	return nil, nil
}
