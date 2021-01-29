package newsfeed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type Options struct {
	URL            string
	UpdateInterval time.Duration
	RetryTime      time.Duration
}

type detector struct {
	creator core.Creator
	options Options
	parser  *gofeed.Parser
}

func (*detector) Close() error {
	return nil
}

func NewDetector(creator core.Creator, options core.Options) core.Detector {
	detectorOptions, ok := options["newsfeed"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid detector options"))
	}

	return &detector{creator: creator, options: detectorOptions, parser: gofeed.NewParser()}
}

// TODO: refactor this function to be reused across different insights instead of copy&pasted
func generateInsight(tx *sql.Tx, c core.Clock, creator core.Creator, content Content) error {
	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.NewsCategory,
		Rating:      core.Unrated,
		ContentType: ContentType,
		Content:     content,
	}

	if err := creator.GenerateInsight(tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type Content struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	Published   time.Time `json:"date_published"`
	GUID        string    `json:"guid"`
}

func (c Content) String() string {
	return c.Description
}

func (c Content) Args() []interface{} {
	return nil
}

func (c Content) TplString() string {
	return c.Description
}

const kind = "newsfeed_last_exec"

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	now := c.Now()

	lastExecTime, err := core.RetrieveLastDetectorExecution(tx, kind)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if !lastExecTime.IsZero() && lastExecTime.Add(d.options.UpdateInterval).After(now) {
		return nil
	}

	timeout := time.Second * 3

	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Info().Msgf("Fetching news insights from %s", d.options.URL)

	parsed, err := d.parser.ParseURLWithContext(d.options.URL, context)
	if err != nil {
		log.Warn().Msgf("Failed to request newfeed insight source %s with error: %v", d.options.URL, err)

		// The request failed. Then try in the next time again
		if err := core.StoreLastDetectorExecution(tx, kind, now.Add(d.options.RetryTime).Add(-d.options.UpdateInterval)); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	for _, item := range parsed.Items {
		if item.PublishedParsed == nil {
			continue
		}

		// TODO: do not scan through all the newsfeed insights, but only through
		// the ones in the time interval from the items in the feed
		alreadyExists, err := insightAlreadyExists(context, tx, item.GUID)
		if err != nil {
			return errorutil.Wrap(err)
		}

		if alreadyExists {
			continue
		}

		if err := generateInsight(tx, c, d.creator, Content{
			Title:       item.Title,
			Description: item.Description,
			Link:        item.Link,
			Published:   *item.PublishedParsed,
			GUID:        item.GUID,
		}); err != nil {
			return errorutil.Wrap(err)
		}
	}

	if err := core.StoreLastDetectorExecution(tx, kind, now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// FIXME: calling this function for each item in a feed (considering a long feed)
// will have almost quadratic execution time on the number of news insights already generated!!!
// It should be optimized to be close to O(n)
// rowserrcheck is not able to notice that query.Err() is called and emits a false positive warning
//nolint:rowserrcheck
func insightAlreadyExists(context context.Context, tx *sql.Tx, guid string) (bool, error) {
	rows, err := tx.QueryContext(context, `select content from insights where content_type = ?`, ContentTypeId)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	defer func() {
		errorutil.MustSucceed(rows.Close())
	}()

	for rows.Next() {
		var rawContent string

		err = rows.Scan(&rawContent)
		if err != nil {
			return false, errorutil.Wrap(err)
		}

		var content Content

		err = json.Unmarshal([]byte(rawContent), &content)
		if err != nil {
			return false, errorutil.Wrap(err)
		}

		if guid == content.GUID {
			return true, nil
		}
	}

	err = rows.Err()
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	return false, nil
}

const (
	ContentType   = "newsfeed_content"
	ContentTypeId = 6
)

func init() {
	core.RegisterContentType(ContentType, ContentTypeId, core.DefaultContentTypeDecoder(&Content{}))
}
