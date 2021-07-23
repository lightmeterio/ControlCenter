// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	"context"
	"errors"
	uuid "github.com/satori/go.uuid"
	"gitlab.com/lightmeter/controlcenter/auth"
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
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"os"
	"path"
	"time"
)

type Workspace struct {
	runner.CancelableRunner
	closeutil.Closers

	deliveries             *deliverydb.DB
	tracker                *tracking.Tracker
	insightsEngine         *insights.Engine
	auth                   *auth.Auth
	rblDetector            *messagerbl.Detector
	rblChecker             localrbl.Checker
	intelCollector         *collector.Collector
	logsLineCountPublisher postfix.Publisher

	dashboard dashboard.Dashboard
	detective detective.Detective
	escalator escalator.Escalator

	NotificationCenter *notification.Center

	settingsMetaHandler *meta.Handler
	settingsRunner      *meta.Runner

	importAnnouncer *announcer.SynchronizingAnnouncer
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
	err = m.Reader.RetrieveJson(context.Background(), meta.UuidMetaKey, &instanceID)

	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
		return nil, errorutil.Wrap(err)
	}

	if errors.Is(err, meta.ErrNoSuchKey) {
		instanceID = uuid.NewV4().String()
		err := m.Writer.StoreJson(context.Background(), meta.UuidMetaKey, instanceID)

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

	intelOptions := intel.Options{
		InstanceID:           instanceID,
		CycleInterval:        time.Second * 30,
		ReportInterval:       time.Minute * 30,
		ReportDestinationURL: "https://intelligence.lightmeter.io/reports",
	}

	intelCollector, logsLineCountPublisher, err := intel.New(workspaceDirectory, deliveries, insightsEngine.Fetcher(), m.Reader, auth, intelOptions)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	logsRunner := newLogsRunner(tracker, deliveries)

	importAnnouncer := announcer.NewSynchronizingAnnouncer(insightsEngine.ImportAnnouncer(), deliveries.MostRecentLogTime, tracker.MostRecentLogTime)

	ws := &Workspace{
		deliveries:             deliveries,
		tracker:                tracker,
		insightsEngine:         insightsEngine,
		auth:                   auth,
		rblDetector:            rblDetector,
		rblChecker:             rblChecker,
		dashboard:              dashboard,
		detective:              messageDetective,
		escalator:              detectiveEscalator,
		settingsMetaHandler:    m,
		settingsRunner:         settingsRunner,
		importAnnouncer:        importAnnouncer,
		intelCollector:         intelCollector,
		logsLineCountPublisher: logsLineCountPublisher,
		Closers: closeutil.New(
			auth,
			tracker,
			deliveries,
			insightsEngine,
			m,
			insightsAccessor,
			intelCollector,
		),
		NotificationCenter: notificationCenter,
	}

	ws.CancelableRunner = runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		// Convert this one here to CancellableRunner!
		ws.rblChecker.StartListening()

		doneInsights, cancelInsights := ws.insightsEngine.Run()
		doneSettings, cancelSettings := ws.settingsRunner.Run()
		doneMsgRbl, cancelMsgRbl := ws.rblDetector.Run()
		doneLogsRunner, cancelLogsRunner := logsRunner.Run()
		doneImporter, cancelImporter := ws.importAnnouncer.Run()
		doneCollector, cancelCollector := intelCollector.Run()

		go func() {
			<-cancel
			cancelLogsRunner()
			cancelMsgRbl()
			cancelSettings()
			cancelInsights()
			cancelImporter()
			cancelCollector()
		}()

		go func() {
			// TODO: handle errors!
			errorutil.MustSucceed(doneLogsRunner())
			errorutil.MustSucceed(doneMsgRbl())
			errorutil.MustSucceed(doneSettings())
			errorutil.MustSucceed(doneInsights())
			errorutil.MustSucceed(doneImporter())
			errorutil.MustSucceed(doneCollector())

			// TODO: return a combination of the "children" errors!
			done <- nil
		}()
	})

	return ws, nil
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

func (ws *Workspace) Detective() detective.Detective {
	return ws.detective
}

func (ws *Workspace) DetectiveEscalationRequester() escalator.Requester {
	return ws.escalator
}

func (ws *Workspace) ImportAnnouncer() announcer.ImportAnnouncer {
	return ws.importAnnouncer
}

func (ws *Workspace) Auth() *auth.Auth {
	return ws.auth
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

	if mostRecentTrackerTime.After(mostRecentDeliverTime) {
		return mostRecentTrackerTime, nil
	}

	return mostRecentDeliverTime, nil
}

func (ws *Workspace) NewPublisher() postfix.Publisher {
	return postfix.ComposedPublisher{ws.tracker.Publisher(), ws.rblDetector.NewPublisher(), ws.logsLineCountPublisher}
}

func (ws *Workspace) HasLogs() bool {
	return ws.deliveries.HasLogs()
}
