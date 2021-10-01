// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	"context"
	"errors"
	uuid "github.com/satori/go.uuid"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/connectionstats"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/detective"
	"gitlab.com/lightmeter/controlcenter/detective/escalator"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights"
	insightsCore "gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/intel"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/messagerbl"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/notification/email"
	"gitlab.com/lightmeter/controlcenter/notification/slack"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/po"
	"gitlab.com/lightmeter/controlcenter/postfixversion"
	"gitlab.com/lightmeter/controlcenter/rawlogsdb"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"os"
	"path"
	"time"
)

type Workspace struct {
	runner.CancellableRunner
	closeutil.Closers

	deliveries              *deliverydb.DB
	rawLogs                 *rawlogsdb.DB
	tracker                 *tracking.Tracker
	connStats               *connectionstats.Stats
	insightsEngine          *insights.Engine
	auth                    *auth.Auth
	rblDetector             *messagerbl.Detector
	rblChecker              localrbl.Checker
	intelCollector          *collector.Collector
	logsLineCountPublisher  postfix.Publisher
	postfixVersionPublisher postfixversion.Publisher

	dashboard dashboard.Dashboard
	detective detective.Detective
	escalator escalator.Escalator

	NotificationCenter *notification.Center

	settingsMetaHandler *metadata.Handler
	settingsRunner      *metadata.SerialWriteRunner

	importAnnouncer         *announcer.SynchronizingAnnouncer
	connectionStatsAccessor *connectionstats.Accessor
	intelAccessor           *collector.Accessor

	databases databases
}

type databases struct {
	closeutil.Closers

	Auth           *dbconn.PooledPair
	Connections    *dbconn.PooledPair
	Insights       *dbconn.PooledPair
	IntelCollector *dbconn.PooledPair
	Logs           *dbconn.PooledPair
	LogTracker     *dbconn.PooledPair
	Master         *dbconn.PooledPair
	RawLogs        *dbconn.PooledPair
}

func newDb(directory string, databaseName string) (*dbconn.PooledPair, error) {
	dbFilename := path.Join(directory, databaseName+".db")
	connPair, err := dbconn.Open(dbFilename, 10)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err := migrator.Run(connPair.RwConn.DB, databaseName); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return connPair, nil
}

type Options struct {
	IsUsingRsyncedLogs bool
	DefaultSettings    metadata.DefaultValues
}

var DefaultOptions = &Options{
	IsUsingRsyncedLogs: false,
	DefaultSettings:    metadata.DefaultValues{},
}

func NewWorkspace(workspaceDirectory string, options *Options) (*Workspace, error) {
	if options == nil {
		options = DefaultOptions
	}

	if err := os.MkdirAll(workspaceDirectory, os.ModePerm); err != nil {
		return nil, errorutil.Wrap(err, "Error creating working directory ", workspaceDirectory)
	}

	allDatabases := databases{Closers: closeutil.New()}

	for _, s := range []struct {
		name string
		db   **dbconn.PooledPair
	}{
		{"auth", &allDatabases.Auth},
		{"connections", &allDatabases.Connections},
		{"insights", &allDatabases.Insights},
		{"intel-collector", &allDatabases.IntelCollector},
		{"logs", &allDatabases.Logs},
		{"logtracker", &allDatabases.LogTracker},
		{"master", &allDatabases.Master},
		{"rawlogs", &allDatabases.RawLogs},
	} {
		db, err := newDb(workspaceDirectory, s.name)

		if err != nil {
			return nil, errorutil.Wrap(err, "Error opening databases in directory ", workspaceDirectory)
		}

		*s.db = db

		allDatabases.Closers.Add(db)
	}

	deliveries, err := deliverydb.New(allDatabases.Logs, &domainmapping.DefaultMapping)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	tracker, err := tracking.New(allDatabases.LogTracker, deliveries.ResultsPublisher())
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	rawLogsDb, err := rawlogsdb.New(allDatabases.RawLogs.RwConn)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	auth, err := auth.NewAuth(allDatabases.Auth, auth.Options{})

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	m, err := metadata.NewDefaultedHandler(allDatabases.Master, options.DefaultSettings)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	settingsRunner := metadata.NewSerialWriteRunner(m)

	// determine instance ID from the database, or create one

	var instanceID string
	err = m.Reader.RetrieveJson(context.Background(), metadata.UuidMetaKey, &instanceID)

	if err != nil && !errors.Is(err, metadata.ErrNoSuchKey) {
		return nil, errorutil.Wrap(err)
	}

	if errors.Is(err, metadata.ErrNoSuchKey) {
		instanceID = uuid.NewV4().String()
		err := m.Writer.StoreJson(context.Background(), metadata.UuidMetaKey, instanceID)

		if err != nil {
			return nil, errorutil.Wrap(err)
		}
	}

	dashboard, err := dashboard.New(allDatabases.Logs.RoConnPool)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	messageDetective, err := detective.New(allDatabases.Logs.RoConnPool)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	detectiveEscalator := escalator.New()

	translators := translator.New(po.DefaultCatalog)

	notificationPolicies := notification.Policies{insights.DefaultNotificationPolicy{}}

	notifiers := map[string]notification.Notifier{
		slack.SettingKey: slack.New(notificationPolicies, m.Reader),
		email.SettingKey: email.New(notificationPolicies, m.Reader),
	}

	policy := &insights.DefaultNotificationPolicy{}

	notificationCenter := notification.New(m.Reader, translators, policy, notifiers)

	rblChecker := localrbl.NewChecker(m.Reader, localrbl.Options{
		NumberOfWorkers:  10,
		Lookup:           localrbl.RealLookup,
		RBLProvidersURLs: localrbl.DefaultRBLs,
	})

	rblDetector := messagerbl.New(globalsettings.New(m.Reader))

	insightsAccessor, err := insights.NewAccessor(allDatabases.Insights)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	insightsEngine, err := insights.NewEngine(
		insightsAccessor,
		notificationCenter,
		insightsOptions(dashboard, rblChecker, rblDetector, detectiveEscalator, allDatabases.Logs.RoConnPool))
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	connStats, err := connectionstats.New(allDatabases.Connections)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	connectionStatsAccessor, err := connectionstats.NewAccessor(allDatabases.Connections.RoConnPool)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	intelOptions := intel.Options{
		InstanceID:           instanceID,
		CycleInterval:        time.Second * 30,
		ReportInterval:       time.Minute * 30,
		ReportDestinationURL: IntelReportDestinationURL,
		IsUsingRsyncedLogs:   options.IsUsingRsyncedLogs,
	}

	intelCollector, logsLineCountPublisher, err := intel.New(
		allDatabases.IntelCollector, allDatabases.Logs.RoConnPool, insightsEngine.Fetcher(),
		m.Reader, auth, allDatabases.Connections.RoConnPool, intelOptions)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	intelAccessor, err := collector.NewAccessor(allDatabases.IntelCollector.RoConnPool)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	logsRunner := newLogsRunner(tracker, deliveries)

	importAnnouncer := announcer.NewSynchronizingAnnouncer(insightsEngine.ImportAnnouncer(), deliveries.MostRecentLogTime, tracker.MostRecentLogTime)

	rblCheckerCancellableRunner := runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		// TODO: Convert this one here to a proper CancellableRunner that can be cancelled...
		rblChecker.StartListening()

		go func() {
			<-cancel
			done <- nil
		}()
	})

	return &Workspace{
		deliveries:              deliveries,
		rawLogs:                 rawLogsDb,
		tracker:                 tracker,
		insightsEngine:          insightsEngine,
		connStats:               connStats,
		auth:                    auth,
		rblDetector:             rblDetector,
		rblChecker:              rblChecker,
		dashboard:               dashboard,
		detective:               messageDetective,
		escalator:               detectiveEscalator,
		settingsMetaHandler:     m,
		settingsRunner:          settingsRunner,
		importAnnouncer:         importAnnouncer,
		intelCollector:          intelCollector,
		intelAccessor:           intelAccessor,
		logsLineCountPublisher:  logsLineCountPublisher,
		postfixVersionPublisher: postfixversion.NewPublisher(settingsRunner.Writer()),
		connectionStatsAccessor: connectionStatsAccessor,
		databases:               allDatabases,
		Closers: closeutil.New(
			connStats,
			deliveries,
			rawLogsDb,
			tracker,
			insightsEngine,
			intelCollector,
			allDatabases,
		),
		NotificationCenter: notificationCenter,
		CancellableRunner: runner.NewCombinedCancellableRunners(
			insightsEngine, settingsRunner, rblDetector, logsRunner, importAnnouncer,
			intelCollector, connStats, rblCheckerCancellableRunner, rawLogsDb),
	}, nil
}

func (ws *Workspace) SettingsAcessors() (*metadata.AsyncWriter, metadata.Reader) {
	return ws.settingsRunner.Writer(), ws.settingsMetaHandler.Reader
}

func (ws *Workspace) InsightsEngine() *insights.Engine {
	return ws.insightsEngine
}

func (ws *Workspace) InsightsFetcher() insightsCore.Fetcher {
	return ws.insightsEngine.Fetcher()
}

func (ws *Workspace) InsightsProgressFetcher() insightsCore.ProgressFetcher {
	return ws.insightsEngine.ProgressFetcher()
}

func (ws *Workspace) Dashboard() dashboard.Dashboard {
	return ws.dashboard
}

func (ws *Workspace) ConnectionStatsAccessor() *connectionstats.Accessor {
	return ws.connectionStatsAccessor
}

func (ws *Workspace) IntelAccessor() *collector.Accessor {
	return ws.intelAccessor
}

func (ws *Workspace) Detective() detective.Detective {
	return ws.detective
}

func (ws *Workspace) DetectiveEscalationRequester() escalator.Requester {
	return ws.escalator
}

func (ws *Workspace) ImportAnnouncer() (announcer.ImportAnnouncer, error) {
	sum, err := ws.MostRecentLogTimeAndSum()
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	// first execution. Must import historical insights
	if sum.Time.IsZero() {
		return ws.importAnnouncer, nil
	}

	// otherwise skip the historical insights import
	return announcer.Skipper(ws.importAnnouncer), nil
}

func (ws *Workspace) Auth() *auth.Auth {
	return ws.auth
}

func (ws *Workspace) MostRecentLogTimeAndSum() (postfix.SumPair, error) {
	mostRecentDeliverTime, err := ws.deliveries.MostRecentLogTime()
	if err != nil {
		return postfix.SumPair{}, errorutil.Wrap(err)
	}

	mostRecentTrackerTime, err := ws.tracker.MostRecentLogTime()
	if err != nil {
		return postfix.SumPair{}, errorutil.Wrap(err)
	}

	mostRecentConnStatsTime, err := ws.connStats.MostRecentLogTime()
	if err != nil {
		return postfix.SumPair{}, errorutil.Wrap(err)
	}

	rawLogsSum, err := rawlogsdb.MostRecentLogTimeAndSum(context.Background(), ws.databases.RawLogs.RoConnPool)
	if err != nil {
		return postfix.SumPair{}, errorutil.Wrap(err)
	}

	// if raw logs has any logs, just use it
	if rawLogsSum.Sum != nil {
		return rawLogsSum, nil
	}

	// In case we have no raw logs unavailable yet (the first execution after rawlogsdb is introduced),
	// try the old way to compute the most recent time from the other databases

	times := [3]time.Time{mostRecentConnStatsTime, mostRecentTrackerTime, mostRecentDeliverTime}

	mostRecent := time.Time{}

	for _, t := range times {
		if t.After(mostRecent) {
			mostRecent = t
		}
	}

	return postfix.SumPair{Time: mostRecent, Sum: nil}, nil
}

func (ws *Workspace) NewPublisher() postfix.Publisher {
	return postfix.ComposedPublisher{
		ws.tracker.Publisher(),
		ws.rblDetector.NewPublisher(),
		ws.logsLineCountPublisher,
		ws.postfixVersionPublisher,
		ws.connStats.Publisher(),
		ws.rawLogs.Publisher(),
	}
}

func (ws *Workspace) HasLogs() bool {
	return ws.deliveries.HasLogs()
}

func (ws *Workspace) RawLogsAccessor() rawlogsdb.Accessor {
	return rawlogsdb.NewAccessor(ws.databases.RawLogs.RoConnPool)
}
