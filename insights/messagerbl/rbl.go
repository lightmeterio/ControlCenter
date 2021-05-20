// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package messagerblinsight

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/messagerbl"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net"
	"time"
)

type Options struct {
	Detector                    messagerbl.Stepper
	MinTimeToGenerateNewInsight time.Duration
}

const (
	ContentType   = "message_rbl"
	ContentTypeId = 5
)

func init() {
	core.RegisterContentType(ContentType, ContentTypeId, core.DefaultContentTypeDecoder(&Content{}))
}

type Content struct {
	Address   net.IP    `json:"address"`
	Recipient string    `json:"recipient"`
	Host      string    `json:"host"`
	Status    string    `json:"delivery_status"`
	Message   string    `json:"message"`
	Time      time.Time `json:"log_time"`
}

func (c Content) Title() notificationCore.ContentComponent {
	return &title{c}
}

func (c Content) Description() notificationCore.ContentComponent {
	return &description{c}
}

func (c Content) Metadata() notificationCore.ContentMetadata {
	return nil
}

type title struct {
	c Content
}

func (t title) String() string {
	return translator.Stringfy(t)
}

func (t title) TplString() string {
	return translator.I18n("IP blocked by %v")
}

func (t title) Args() []interface{} {
	return []interface{}{t.c.Host}
}

type description struct {
	c Content
}

func (d description) String() string {
	return translator.Stringfy(d)
}

func (d description) TplString() string {
	return translator.I18n("The IP %v cannot deliver to %v (%v)")
}

func (d description) Args() []interface{} {
	return []interface{}{d.c.Address, d.c.Recipient, d.c.Host}
}

func (c Content) HelpLink(urlContainer core.URLContainer) string {
	return urlContainer.Get(ContentType + "_" + c.Host)
}

type detector struct {
	options Options
	creator core.Creator
}

func (detector) IsHistoricalDetector() {
	// Really empty, just to implement the HistoricalDetector interface
}

func NewDetector(creator core.Creator, options core.Options) core.Detector {
	detectorOptions, ok := options["messagerbl"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid detector options"))
	}

	return &detector{
		options: detectorOptions,
		creator: creator,
	}
}

func maybeAddNewInsightFromMessage(d *detector, r messagerbl.Result, c core.Clock, tx *sql.Tx) error {
	detectionKind := fmt.Sprintf("message_rbl_%s", r.Host)

	t, err := core.RetrieveLastDetectorExecution(tx, detectionKind)
	if err != nil {
		return errorutil.Wrap(err)
	}

	// During historical import, the "fake" clock might not yet have "reached" the clock used to generate the
	// insight, so we use the log time instead, to prevent this insight to be generated in the past
	// Ref #493
	insightTime := r.Time

	// MinTimeToGenerateNewInsight
	if t.Add(d.options.MinTimeToGenerateNewInsight).After(insightTime) {
		log.Info().Msgf("Ignoring RBL insight for host %v that has been generated %v ago", r.Host, insightTime.Sub(t))
		return nil
	}

	content := Content{
		Address:   d.options.Detector.IPAddress(context.Background()),
		Message:   r.Payload.ExtraMessage,
		Recipient: r.Payload.RecipientDomainPart,
		Status:    r.Payload.Status.String(),
		Host:      r.Host,
		Time:      r.Time,
	}

	if err := generateInsight(tx, insightTime, d.creator, content); err != nil {
		return errorutil.Wrap(err)
	}

	if err := core.StoreLastDetectorExecution(tx, detectionKind, insightTime); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	return d.options.Detector.Step(func(allResults []messagerbl.Results) error {
		for _, results := range allResults {
			for i := 0; i < results.Size; i++ {
				r := results.Values[i]
				if err := maybeAddNewInsightFromMessage(d, r, c, tx); err != nil {
					return errorutil.Wrap(err)
				}
			}
		}

		return nil
	})
}

func (d *detector) Close() error {
	return nil
}

func generateInsight(tx *sql.Tx, insightTime time.Time, creator core.Creator, content Content) error {
	properties := core.InsightProperties{
		Time:        insightTime,
		Category:    core.LocalCategory,
		Rating:      core.BadRating,
		ContentType: ContentType,
		Content:     content,
	}

	if err := creator.GenerateInsight(context.Background(), tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
