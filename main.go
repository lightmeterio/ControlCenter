// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"
	"gitlab.com/lightmeter/controlcenter/config"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/dirlogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/dirwatcher"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	"gitlab.com/lightmeter/controlcenter/logeater/socketsource"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/server"
	"gitlab.com/lightmeter/controlcenter/subcommand"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/version"
	"gitlab.com/lightmeter/controlcenter/workspace"
)

func main() {
	conf, err := config.Parse(os.Args[1:], os.LookupEnv)
	if err != nil {
		errorutil.Dief(conf.Verbose, errorutil.Wrap(err), "Could not parse command-line arguments or environment variables")
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().Str("service", "controlcenter").Str("instanceid", uuid.NewV4().String()).Caller().Logger()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if conf.Verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if conf.ShowVersion {
		version.PrintVersion()
		return
	}

	liabilityDisclamer := `This program comes with ABSOLUTELY NO WARRANTY. This is free software, and you are welcome to redistribute it under certain conditions; see here for details: https://lightmeter.io/lmcc-license.`

	log.Info().Msg(liabilityDisclamer)

	lmsqlite3.Initialize(lmsqlite3.Options{})

	if conf.MigrateDownToOnly {
		subcommand.PerformMigrateDownTo(conf.Verbose, conf.WorkspaceDirectory, conf.MigrateDownToDatabaseName, int64(conf.MigrateDownToVersion))
		return
	}

	if len(conf.EmailToPasswdReset) > 0 {
		subcommand.PerformPasswordReset(conf.Verbose, conf.WorkspaceDirectory, conf.EmailToPasswdReset, conf.PasswordToReset)
		return
	}

	ws, err := workspace.NewWorkspace(conf.WorkspaceDirectory)

	if err != nil {
		errorutil.Dief(conf.Verbose, errorutil.Wrap(err), "Error creating / opening workspace directory for storing application files: %s. Try specifying a different directory (using -workspace), or check you have permission to write to the specified location.", conf.WorkspaceDirectory)
	}

	logSource, err := buildLogSource(ws, conf)

	if err != nil {
		errorutil.Dief(conf.Verbose, err, "Error setting up logs reading")
	}

	done, cancel := ws.Run()

	logReader := logsource.NewReader(logSource, ws.NewPublisher())

	// only import logs and exit when they end. Does not start web server.
	// It's useful for benchmarking importing logs.
	if conf.ImportOnly {
		err := logReader.Run()

		if err != nil {
			errorutil.Dief(conf.Verbose, err, "Error reading logs")
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
		errorutil.Dief(conf.Verbose, err, "Error: Workspace execution has ended, which should never happen here!")
	}()

	go func() {
		err := logReader.Run()
		if err != nil {
			errorutil.Dief(conf.Verbose, err, "Error reading logs")
		}
	}()

	httpServer := server.HttpServer{
		Workspace:          ws,
		WorkspaceDirectory: conf.WorkspaceDirectory,
		Timezone:           conf.Timezone,
		Address:            conf.Address,
	}

	errorutil.MustSucceed(httpServer.Start(), "server died")
}

func importAnnouncerOnlyForFirstExecution(initialTime time.Time, a announcer.ImportAnnouncer) announcer.ImportAnnouncer {
	// first execution. Must import historical insights
	if initialTime.IsZero() {
		return a
	}

	// otherwise skip the historical insights import
	return announcer.Skipper(a)
}

func buildLogSource(ws *workspace.Workspace, conf config.Config) (logsource.Source, error) {
	mostRecentTime, err := ws.MostRecentLogTime()
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	announcer := importAnnouncerOnlyForFirstExecution(mostRecentTime, ws.ImportAnnouncer())

	patterns := func(patterns []string) dirwatcher.LogPatterns {
		if len(patterns) == 0 {
			return dirwatcher.DefaultLogPatterns
		}

		return dirwatcher.BuildLogPatterns(patterns)
	}(conf.LogPatterns)

	if len(conf.DirToWatch) > 0 {
		s, err := dirlogsource.New(conf.DirToWatch, mostRecentTime, announcer, !conf.ImportOnly, conf.RsyncedDir, conf.LogFormat, patterns)
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

	errorutil.Dief(conf.Verbose, nil, "No logs sources specified or import flag provided! Use -help to more info.")

	return nil, nil
}
