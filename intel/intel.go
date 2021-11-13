// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package intel

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/auth"
	insightscore "gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/intel/bruteforce"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	intelConnectionStats "gitlab.com/lightmeter/controlcenter/intel/connectionstats"
	"gitlab.com/lightmeter/controlcenter/intel/core"
	"gitlab.com/lightmeter/controlcenter/intel/insights"
	"gitlab.com/lightmeter/controlcenter/intel/logslinecount"
	"gitlab.com/lightmeter/controlcenter/intel/mailactivity"
	"gitlab.com/lightmeter/controlcenter/intel/receptor"
	"gitlab.com/lightmeter/controlcenter/intel/topdomains"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/postfixversion"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/version"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type SchedFileReader func() (io.ReadCloser, error)

var ErrFailedReadingSchedFile = errors.New(`Failed reading /proc/1/sched file`)

func isSchedFileContentInsideContainer(r SchedFileReader) (insideContainer bool, err error) {
	f, err := r()
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	defer func() {
		// TODO: rewrite this code to reuse the UpdateError() once it's merged!!!
		if cErr := f.Close(); cErr != nil && err == nil {
			err = cErr
		}
	}()

	scanner := bufio.NewScanner(f)

	if !scanner.Scan() {
		return false, errorutil.Wrap(ErrFailedReadingSchedFile)
	}

	line := scanner.Text()

	index := strings.Index(line, " ")
	if index == -1 {
		return false, errorutil.Wrap(ErrFailedReadingSchedFile)
	}

	if err := scanner.Err(); err != nil {
		return false, errorutil.Wrap(ErrFailedReadingSchedFile)
	}

	processName := line[0:index]

	return processName == "lightmeter", nil
}

func DefaultSchedFileReader() (io.ReadCloser, error) {
	return os.Open("/proc/1/sched")
}

type Metadata struct {
	InstanceID         string  `json:"instance_id"`
	LocalIP            *string `json:"postfix_public_ip,omitempty"`
	PublicURL          *string `json:"public_url,omitempty"`
	UserEmail          *string `json:"user_email,omitempty"`
	PostfixVersion     *string `json:"postfix_version,omitempty"`
	MailKind           *string `json:"mail_kind,omitempty"`
	IsDockerContainer  bool    `json:"is_docker_container"`
	IsUsingRsyncedLogs bool    `json:"is_using_rsynced_logs"`
}

type Version struct {
	Version     string `json:"version"`
	TagOrBranch string `json:"tag_or_branch"`
	Commit      string `json:"commit"`
}

type ReportWithMetadata struct {
	Metadata Metadata         `json:"metadata"`
	Version  Version          `json:"app_version"`
	Payload  collector.Report `json:"payload"`
}

type Dispatcher struct {
	InstanceID           string
	VersionBuilder       func() Version
	ReportDestinationURL string
	SettingsReader       metadata.Reader
	Auth                 auth.Registrar
	SchedFileReader      SchedFileReader
	IsUsingRsyncedLogs   bool
}

func (d *Dispatcher) Dispatch(r collector.Report) error {
	log.Info().Msgf("Sending a new Network intelligence report in the interval %v and with %v rows", r.Interval, len(r.Content))

	metadata, err := func() (Metadata, error) {
		// InstanceID is always available
		metadata := Metadata{InstanceID: d.InstanceID}

		// there can be a postfix version even with no user registered
		metadata.PostfixVersion = d.getPostfixVersion()

		userData, err := d.Auth.GetFirstUser(context.Background())
		if err != nil && !errors.Is(err, auth.ErrNoUser) { // if no user is registered, simply don't send any email
			return metadata, errorutil.Wrap(err)
		}

		if err == nil {
			metadata.UserEmail = &userData.Email
		}

		metadata.PublicURL, metadata.LocalIP, metadata.MailKind = d.getGlobalSettings()

		insideContainer, err := isSchedFileContentInsideContainer(d.SchedFileReader)
		if err != nil {
			return metadata, errorutil.Wrap(err)
		}

		metadata.IsDockerContainer = insideContainer

		metadata.IsUsingRsyncedLogs = d.IsUsingRsyncedLogs

		return metadata, nil
	}()

	if err != nil {
		return errorutil.Wrap(err)
	}

	reportWithMetadata := ReportWithMetadata{
		Version:  d.VersionBuilder(),
		Metadata: metadata,
		Payload:  r,
	}

	json, err := json.Marshal(reportWithMetadata)
	if err != nil {
		return errorutil.Wrap(err)
	}

	ctx, cancelCtx := context.WithTimeout(context.Background(), 2500*time.Millisecond)

	defer cancelCtx()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.ReportDestinationURL, bytes.NewBuffer(json))
	if err != nil {
		return errorutil.Wrap(err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}

	response, err := client.Do(req)
	if err != nil {
		log.Err(err).Msgf("Could not send report")

		// NOTE: a network error is not a hard error
		// TODO: maybe retry until it succeeds?
		return nil
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			// Not a fatal error; just ignore it
			log.Err(err).Msgf("Error closing response body")
		}
	}()

	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Err(err).Msgf("Could not send report")
			return nil
		}

		log.Error().Msgf(`Request failed with error code %v and error: %v`, response.Status, string(body))
	}

	return nil
}

func (d *Dispatcher) getGlobalSettings() (*string, *string, *string) {
	settings, err := globalsettings.GetSettings(context.Background(), d.SettingsReader)

	if err != nil && errors.Is(err, metadata.ErrNoSuchKey) {
		log.Warn().Msgf("Unexpected error retrieving global settings")
	}

	if err != nil {
		return nil, nil, nil
	}

	addr := func(s string) *string {
		return &s
	}

	publicURL := func() *string {
		if settings.PublicURL != "" {
			return addr(settings.PublicURL)
		}

		return nil
	}()

	localIP := func() *string {
		if settings.LocalIP.IP != nil {
			return addr(settings.LocalIP.String())
		}

		return nil
	}()

	mailKind := func() *string {
		mailKind, err := d.SettingsReader.Retrieve(context.Background(), "mail_kind")

		if err != nil && !errors.Is(err, metadata.ErrNoSuchKey) {
			log.Warn().Msgf("Unexpected error retrieving mail_kind")
		}

		if err != nil {
			return nil
		}

		s, ok := mailKind.(string)

		if !ok {
			log.Warn().Msgf("mail_kind couldn't be cast to string")
			return nil
		}

		return &s
	}()

	return publicURL, localIP, mailKind
}

func (d *Dispatcher) getPostfixVersion() *string {
	var version string
	err := d.SettingsReader.RetrieveJson(context.Background(), postfixversion.SettingKey, &version)

	if err != nil && !errors.Is(err, metadata.ErrNoSuchKey) {
		log.Warn().Msgf("Unexpected error retrieving postfix version")
	}

	if err != nil {
		return nil
	}

	return &version
}

type Options struct {
	InstanceID string

	// How often should the intel loop should run
	CycleInterval time.Duration

	// How often should the reports be dispatched/sent?
	ReportInterval time.Duration

	ReportDestinationURL string

	// whether the postfix logs are being received via rsync
	IsUsingRsyncedLogs bool
}

func DefaultVersionBuilder() Version {
	return Version{Version: version.Version, TagOrBranch: version.TagOrBranch, Commit: version.Commit}
}

func New(intelDb *dbconn.PooledPair, deliveryDbPool *dbconn.RoPool, fetcher insightscore.Fetcher,
	settingsReader metadata.Reader, auth *auth.Auth, connStatsPool *dbconn.RoPool,
	options Options) (*Runner, *logslinecount.Publisher, bruteforce.Checker, error) {
	logslinePublisher := logslinecount.NewPublisher()

	reporters := collector.Reporters{
		mailactivity.NewReporter(deliveryDbPool),
		insights.NewReporter(fetcher),
		logslinecount.NewReporter(logslinePublisher),
		topdomains.NewReporter(deliveryDbPool),
		intelConnectionStats.NewReporter(connStatsPool),
	}

	coreOptions := core.Options{
		CycleInterval:  options.CycleInterval,
		ReportInterval: options.ReportInterval,
	}

	dispatcher := &Dispatcher{
		InstanceID:           options.InstanceID,
		VersionBuilder:       DefaultVersionBuilder,
		SettingsReader:       settingsReader,
		ReportDestinationURL: options.ReportDestinationURL,
		Auth:                 auth,
		SchedFileReader:      DefaultSchedFileReader,
		IsUsingRsyncedLogs:   options.IsUsingRsyncedLogs,
	}

	dbRunner := core.NewRunner(intelDb.RwConn, coreOptions)

	c, err := collector.New(dbRunner.Actions, coreOptions, reporters, dispatcher)
	if err != nil {
		return nil, nil, nil, errorutil.Wrap(err)
	}

	receptorOptions := receptor.Options{
		PollInterval: 1 * time.Minute,
		InstanceID:   options.InstanceID,
	}

	r, err := receptor.New(dbRunner.Actions, intelDb.RoConnPool, &requester{}, receptorOptions)
	if err != nil {
		return nil, nil, nil, errorutil.Wrap(err)
	}

	return &Runner{
		Closers:           closeutil.New(c, r),
		CancellableRunner: runner.NewDependantPairCancellableRunner(runner.NewCombinedCancellableRunners(c, r), dbRunner),
	}, logslinePublisher, r, nil
}

type Runner struct {
	closeutil.Closers
	runner.CancellableRunner
}

type requester struct {
}

func (r *requester) Request(context.Context, receptor.Payload) (*receptor.Event, error) {
	return nil, nil
}
