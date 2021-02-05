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
	core.RegisterContentType(ContentType, ContentTypeId, core.DefaultContentTypeDecoder(&content{}))
}

type content struct {
	Address   net.IP    `json:"address"`
	Recipient string    `json:"recipient"`
	Host      string    `json:"host"`
	Status    string    `json:"delivery_status"`
	Message   string    `json:"message"`
	Time      time.Time `json:"log_time"`
}

func (c content) String() string {
	return translator.Stringfy(c)
}

func (c content) TplString() string {
	return translator.I18n("The IP %v cannot deliver to %v (%v)")
}

func (c content) Args() []interface{} {
	return []interface{}{c.Address, c.Recipient, c.Host}
}

func (c content) HelpLink(urlContainer core.URLContainer) string {
	return urlContainer.Get(ContentType + "_" + c.Host)
}

type detector struct {
	options Options
	creator core.Creator
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

	now := c.Now()

	// Don't do anything if there's already an insight for such host in the past
	// MinTimeToGenerateNewInsight
	if t.Add(d.options.MinTimeToGenerateNewInsight).After(now) {
		log.Info().Msgf("Ignoring RBL insight for host %v that has been generated %v ago", r.Host, now.Sub(t))
		return nil
	}

	content := content{
		Address:   d.options.Detector.IPAddress(context.Background()),
		Message:   r.Payload.ExtraMessage,
		Recipient: r.Payload.RecipientDomainPart,
		Status:    r.Payload.Status.String(),
		Host:      r.Host,
		Time:      r.Time,
	}

	if err := generateInsight(tx, c, d.creator, content); err != nil {
		return errorutil.Wrap(err)
	}

	if err := core.StoreLastDetectorExecution(tx, detectionKind, now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	return d.options.Detector.Step(func(r messagerbl.Result) error {
		return maybeAddNewInsightFromMessage(d, r, c, tx)
	}, func() error {
		return nil
	})
}

func (d *detector) Close() error {
	return nil
}

// TODO: refactor this function to be reused across different insights instead of copy&pasted
func generateInsight(tx *sql.Tx, c core.Clock, creator core.Creator, content content) error {
	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.BadRating,
		ContentType: ContentType,
		Content:     content,
	}

	if err := creator.GenerateInsight(tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
