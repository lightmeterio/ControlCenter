// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights"
	insightsCore "gitlab.com/lightmeter/controlcenter/insights/core"
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

	deliveries     *deliverydb.DB
	tracker        *tracking.Tracker
	insightsEngine *insights.Engine
	auth           *auth.Auth
	rblDetector    *messagerbl.Detector
	rblChecker     localrbl.Checker

	dashboard dashboard.Dashboard

	NotificationCenter *notification.Center

	settingsMetaHandler *meta.Handler
	settingsRunner      *meta.Runner

	importAnnouncer *announcer.SynchronizingAnnouncer

	closes closeutil.Closers
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

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	dashboard, err := dashboard.New(deliveries.ConnPool())

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

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

	insightsAcessor, err := insights.NewAccessor(workspaceDirectory)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	insightsEngine, err := insights.NewEngine(insightsAcessor, notificationCenter, insightsOptions(dashboard, rblChecker, rblDetector))
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	logsRunner := newLogsRunner(tracker, deliveries)

	importAnnouncer := announcer.NewSynchronizingAnnouncer(insightsEngine.ImportAnnouncer(), deliveries.MostRecentLogTime, tracker.MostRecentLogTime)

	ws := &Workspace{
		deliveries:          deliveries,
		tracker:             tracker,
		insightsEngine:      insightsEngine,
		auth:                auth,
		rblDetector:         rblDetector,
		rblChecker:          rblChecker,
		dashboard:           dashboard,
		settingsMetaHandler: m,
		settingsRunner:      settingsRunner,
		importAnnouncer:     importAnnouncer,
		closes: closeutil.New(
			auth,
			tracker,
			deliveries,
			insightsEngine,
			m,
			insightsAcessor,
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

		// We don't need to explicitly ends the importer, as it'll
		// end when the import process finished, as it's a single-shot process!
		doneImporter, _ := ws.importAnnouncer.Run()

		go func() {
			<-cancel
			cancelLogsRunner()
			cancelMsgRbl()
			cancelSettings()
			cancelInsights()
		}()

		go func() {
			// TODO: handle errors!
			errorutil.MustSucceed(doneLogsRunner())
			errorutil.MustSucceed(doneMsgRbl())
			errorutil.MustSucceed(doneSettings())
			errorutil.MustSucceed(doneInsights())
			errorutil.MustSucceed(doneImporter())

			// TODO: return a combination of the "children" errors!
			done <- nil
		}()
	})

	return ws, nil
}

func (ws *Workspace) SettingsAcessors() (*meta.AsyncWriter, *meta.Reader) {
	return ws.settingsRunner.Writer(), ws.settingsMetaHandler.Reader
}

func (ws *Workspace) InsightsFetcher() insightsCore.Fetcher {
	return ws.insightsEngine.Fetcher()
}

func (ws *Workspace) Dashboard() dashboard.Dashboard {
	return ws.dashboard
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
	return postfix.ComposedPublisher{ws.tracker.Publisher(), ws.rblDetector.NewPublisher()}
}

func (ws *Workspace) Close() error {
	return ws.closes.Close()
}

func (ws *Workspace) HasLogs() bool {
	return ws.deliveries.HasLogs()
}
