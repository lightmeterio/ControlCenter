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
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/messagerbl"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/notification/email"
	"gitlab.com/lightmeter/controlcenter/notification/slack"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/po"
	"gitlab.com/lightmeter/controlcenter/postfixversion"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"os"
	"path"
	"sort"
	"time"
)

type Workspace struct {
	runner.CancellableRunner
	closeutil.Closers

	deliveries              *deliverydb.DB
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

	settingsMetaHandler *meta.Handler
	settingsRunner      *meta.Runner

	importAnnouncer         *announcer.SynchronizingAnnouncer
	connectionStatsAccessor *connectionstats.Accessor
}

func NewWorkspace(workspaceDirectory string) (*Workspace, error) {
	if err := os.MkdirAll(workspaceDirectory, os.ModePerm); err != nil {
		return nil, errorutil.Wrap(err, "Error creating working directory ", workspaceDirectory)
	}

	deliveries, err := deliverydb.New(workspaceDirectory, &domainmapping.DefaultMapping)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	tracker, err := tracking.New(workspaceDirectory, deliveries.ResultsPublisher())
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	auth, err := auth.NewAuth(workspaceDirectory, auth.Options{})

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	metadataConnPair, err := dbconn.Open(path.Join(workspaceDirectory, "master.db"), 5)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	m, err := meta.NewHandler(metadataConnPair, "master")

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	settingsRunner := meta.NewRunner(m)

	// determine instance ID from the database, or create one

	var instanceID string
	err = m.Reader.RetrieveJson(context.Background(), intel.SettingKey, &instanceID)

	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return nil, errorutil.Wrap(err)
	}

	if errors.Is(err, meta.ErrNoSuchKey) {
		instanceID = uuid.NewV4().String()
		err := m.Writer.StoreJson(context.Background(), intel.SettingKey, instanceID)

		if err != nil {
			return nil, errorutil.Wrap(err)
		}
	}

	dashboard, err := dashboard.New(deliveries.ConnPool())

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	messageDetective, err := detective.New(deliveries.ConnPool())

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

	insightsAccessor, err := insights.NewAccessor(workspaceDirectory)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	insightsEngine, err := insights.NewEngine(
		insightsAccessor,
		notificationCenter,
		insightsOptions(dashboard, rblChecker, rblDetector, detectiveEscalator))
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	connStats, err := connectionstats.New(workspaceDirectory)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	connectionStatsAccessor, err := connectionstats.NewAccessor(connStats.ConnPool())
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	intelOptions := intel.Options{
		InstanceID:           instanceID,
		CycleInterval:        time.Second * 30,
		ReportInterval:       time.Minute * 30,
		ReportDestinationURL: IntelReportDestinationURL,
	}

	intelCollector, logsLineCountPublisher, err := intel.New(
		workspaceDirectory, deliveries, insightsEngine.Fetcher(),
		m.Reader, auth, connStats, intelOptions)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	logsRunner := newLogsRunner(tracker, deliveries)

	importAnnouncer := announcer.NewSynchronizingAnnouncer(insightsEngine.ImportAnnouncer(), deliveries.MostRecentLogTime, tracker.MostRecentLogTime)

	return &Workspace{
		deliveries:              deliveries,
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
		logsLineCountPublisher:  logsLineCountPublisher,
		postfixVersionPublisher: postfixversion.NewPublisher(settingsRunner.Writer()),
		connectionStatsAccessor: connectionStatsAccessor,
		Closers: closeutil.New(
			auth,
			tracker,
			deliveries,
			insightsEngine,
			m,
			insightsAccessor,
			intelCollector,
			connStats,
		),
		NotificationCenter: notificationCenter,
		CancellableRunner: runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
			// Convert this one here to CancellableRunner!
			rblChecker.StartListening()

			doneAll, cancelAll := runner.Run(
				insightsEngine, settingsRunner, rblDetector,
				logsRunner, importAnnouncer, intelCollector, connStats,
			)

			go func() {
				<-cancel
				cancelAll()
			}()

			go func() {
				if err := doneAll(); err != nil {
					done <- errorutil.Wrap(err)
					return
				}

				done <- nil
			}()
		}),
	}, nil
}

func (ws *Workspace) SettingsAcessors() (*meta.AsyncWriter, *meta.Reader) {
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

func (ws *Workspace) Detective() detective.Detective {
	return ws.detective
}

func (ws *Workspace) DetectiveEscalationRequester() escalator.Requester {
	return ws.escalator
}

func (ws *Workspace) ImportAnnouncer() (announcer.ImportAnnouncer, error) {
	mostRecentTime, err := ws.MostRecentLogTime()
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	// first execution. Must import historical insights
	if mostRecentTime.IsZero() {
		return ws.importAnnouncer, nil
	}

	// otherwise skip the historical insights import
	return announcer.Skipper(ws.importAnnouncer), nil
}

func (ws *Workspace) Auth() *auth.Auth {
	return ws.auth
}

type mostRecentTimes [3]time.Time

func (t mostRecentTimes) Len() int {
	return len(t)
}

func (t mostRecentTimes) Less(i, j int) bool {
	return t[i].Before(t[j])
}

func (t mostRecentTimes) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (ws *Workspace) MostRecentLogTime() (time.Time, error) {
	mostRecentDeliverTime, err := ws.deliveries.MostRecentLogTime()
	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	mostRecentTrackerTime, err := ws.tracker.MostRecentLogTime()
	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	mostRecentConnStatsTime, err := ws.connStats.MostRecentLogTime()
	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	times := mostRecentTimes{mostRecentConnStatsTime, mostRecentTrackerTime, mostRecentDeliverTime}

	sort.Sort(times)

	return times[len(times)-1], nil
}

func (ws *Workspace) NewPublisher() postfix.Publisher {
	return postfix.ComposedPublisher{
		ws.tracker.Publisher(),
		ws.rblDetector.NewPublisher(),
		ws.logsLineCountPublisher,
		ws.postfixVersionPublisher,
		ws.connStats.Publisher(),
	}
}

func (ws *Workspace) HasLogs() bool {
	return ws.deliveries.HasLogs()
}
