// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"math"
	"time"
)

type Category int

func (c Category) String() string {
	switch c {
	case LocalCategory:
		return translator.I18n("local")
	case ComparativeCategory:
		return translator.I18n("comparative")
	case NewsCategory:
		return translator.I18n("news")
	case IntelCategory:
		return translator.I18n("intel")
	case ArchivedCategory:
		return translator.I18n("archived")
	case NoCategory:
		fallthrough
	default:
		log.Panic().Msgf("Invalid category: %d", int(c))
		return ""
	}
}

func (Category) Args() []interface{} {
	return nil
}

func (c Category) TplString() string {
	return c.String()
}

const (
	NoCategory          Category = 0
	LocalCategory       Category = 1
	NewsCategory        Category = 2
	ComparativeCategory Category = 3
	IntelCategory       Category = 4
	ArchivedCategory    Category = 5
)

func BuildCategoryByName(n string) Category {
	switch n {
	case "local":
		return LocalCategory
	case "comparative":
		return ComparativeCategory
	case "news":
		return NewsCategory
	case "intel":
		return IntelCategory
	case "archived":
		return ArchivedCategory
	default:
		return NoCategory
	}
}

func BuildFilterByName(n string) FetchFilter {
	switch n {
	case "category":
		return FilterByCategory
	default:
		return NoFetchFilter
	}
}

func BuildOrderByName(n string) FetchOrder {
	switch n {
	case "creationAsc":
		return OrderByCreationAsc
	case "creationDesc":
		return OrderByCreationDesc
	default:
		return OrderByCreationDesc
	}
}

type Rating int

func (r Rating) String() string {
	switch r {
	case BadRating:
		return "bad"
	case OkRating:
		return "ok"
	case GoodRating:
		return "good"
	case Unrated:
		return "unrated"
	default:
		log.Panic().Msgf("Invalid/Unknown rating value: %d", int(r))
		return ""
	}
}

func (Rating) Args() []interface{} {
	return nil
}

func (r Rating) TplString() string {
	return r.String()
}

// The rating values are spaced in order to allow newer values to be added between existing ones
// without requiring data migration, as such values are stored in the insights database.
const (
	// NOTE: the Unrated value is a bit peculiar/special and don't really fit any order.
	// For instance, should listing "all insights with are ok or lower" return insights with no rating?
	// If yes, the query should explicitly remove Unrated insights.
	// In "non sql" code, rating is an optional value, and the "empty" value corresponds to Unrated.
	Unrated Rating = 0

	BadRating  Rating = 100
	OkRating   Rating = 200
	GoodRating Rating = 300
)

type FetchedInsight interface {
	ID() int
	Time() time.Time
	Category() Category
	Rating() Rating
	Content() Content
	ContentType() string
}

type FetchFilter int

const (
	NoFetchFilter FetchFilter = iota
	FilterByCategory
)

type FetchOrder int

const (
	OrderByCreationDesc FetchOrder = iota
	OrderByCreationAsc
)

type FetchOptions struct {
	Interval   timeutil.TimeInterval
	FilterBy   FetchFilter
	OrderBy    FetchOrder
	MaxEntries int
	Category   Category
}

type Fetcher interface {
	FetchInsights(context.Context, FetchOptions) ([]FetchedInsight, error)
}

type queryKey struct {
	order  FetchOrder
	filter FetchFilter
}

type paramBuilder func(FetchOptions) []interface{}

type queryValue struct {
	p paramBuilder
}

type fetcher struct {
	pool    *dbconn.RoPool
	queries map[queryKey]queryValue
}

func buildSelectStmt(where, order string) string {
	return fmt.Sprintf(`
	with
	non_archived(id, time, category, rating, content_type, content) as (
		select
			insights.rowid, insights.time, insights.category, insights.rating, insights.content_type, insights.content
		from
			insights left join insights_status on insights.rowid = insights_status.insight_id
		where
			insights_status.id is null
	),
	archived(id, time, category, rating, content_type, content) as (
		select
			insights.rowid, insights.time, insights_status.status, insights.rating, insights.content_type, insights.content
		from
			insights join insights_status on insights.rowid = insights_status.insight_id
		where insights_status.status = %d
	),
	united(id, time, category, rating, content_type, content) as (
		select * from non_archived union select * from archived
	)
	select * from united
	where %s
	order by %s
	limit ?
	`, int(ArchivedCategory), where, order)
}

func buildLimitForFetchOptions(o FetchOptions) int {
	if o.MaxEntries == 0 {
		return math.MaxInt32
	}

	return o.MaxEntries
}

func NewFetcher(pool *dbconn.RoPool) (Fetcher, error) {
	buildQuery := func(key queryKey, s string) error {
		err := pool.ForEach(func(c *dbconn.RoPooledConn) error {
			//nolint:sqlclosecheck
			q, err := c.Prepare(s)
			if err != nil {
				return errorutil.Wrap(err)
			}

			c.Stmts[key] = q

			c.Closers.Add(q)

			return nil
		})

		if err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	type queriesBuilderPair struct {
		key          queryKey
		value        func(key queryKey) error
		paramBuilder paramBuilder
	}

	queriesBuilders := []queriesBuilderPair{
		{
			key: queryKey{order: OrderByCreationDesc, filter: NoFetchFilter},
			value: func(key queryKey) error {
				return buildQuery(key, buildSelectStmt(`time between ? and ? and category != ?`, `time desc`))
			},
			paramBuilder: func(o FetchOptions) []interface{} {
				return []interface{}{o.Interval.From.Unix(), o.Interval.To.Unix(), ArchivedCategory, buildLimitForFetchOptions(o)}
			},
		},
		{
			key: queryKey{order: OrderByCreationDesc, filter: FilterByCategory},
			value: func(key queryKey) error {
				return buildQuery(key, buildSelectStmt(`time between ? and ? and category = ?`, `time desc`))
			},
			paramBuilder: func(o FetchOptions) []interface{} {
				return []interface{}{o.Interval.From.Unix(), o.Interval.To.Unix(), o.Category, buildLimitForFetchOptions(o)}
			},
		},
		{
			key: queryKey{order: OrderByCreationAsc, filter: FilterByCategory},
			value: func(key queryKey) error {
				return buildQuery(key, buildSelectStmt(`time between ? and ? and category = ?`, `time asc`))
			},
			paramBuilder: func(o FetchOptions) []interface{} {
				return []interface{}{o.Interval.From.Unix(), o.Interval.To.Unix(), o.Category, buildLimitForFetchOptions(o)}
			},
		},
		{
			key: queryKey{order: OrderByCreationAsc, filter: NoFetchFilter},
			value: func(key queryKey) error {
				return buildQuery(key, buildSelectStmt(`time between ? and ? and category != ?`, `time asc`))
			},
			paramBuilder: func(o FetchOptions) []interface{} {
				return []interface{}{o.Interval.From.Unix(), o.Interval.To.Unix(), ArchivedCategory, buildLimitForFetchOptions(o)}
			},
		},
	}

	queries := map[queryKey]queryValue{}

	for _, b := range queriesBuilders {
		if err := b.value(b.key); err != nil {
			return nil, errorutil.Wrap(err)
		}

		queries[b.key] = queryValue{p: b.paramBuilder}
	}

	return &fetcher{queries: queries, pool: pool}, nil
}

type fetchedInsight struct {
	id          int
	time        time.Time
	rating      Rating
	category    Category
	contentType string
	content     Content
}

func (f *fetchedInsight) ID() int {
	return f.id
}

func (f *fetchedInsight) Time() time.Time {
	return f.time
}

func (f *fetchedInsight) Category() Category {
	return f.category
}

func (f *fetchedInsight) Rating() Rating {
	return f.rating
}

func (f *fetchedInsight) ContentType() string {
	return f.contentType
}

func (f *fetchedInsight) Content() Content {
	return f.content
}

// rowserrcheck is not able to notice that query.Err() is called and emits a false positive warning
//nolint:rowserrcheck
func (f *fetcher) FetchInsights(ctx context.Context, options FetchOptions) ([]FetchedInsight, error) {
	conn, release := f.pool.Acquire()

	defer release()

	key := queryKey{order: options.OrderBy, filter: options.FilterBy}

	stmt, ok := conn.Stmts[key]
	if !ok {
		log.Panic().Msgf("Sql stmt for options %v not implemented!!!!", options)
	}

	query := f.queries[key]

	if !ok {
		log.Panic().Msgf("Sql arguments for options %v not implemented!!!!", options)
	}

	rows, err := stmt.QueryContext(ctx, query.p(options)...)

	if err != nil {
		return []FetchedInsight{}, errorutil.Wrap(err)
	}

	defer func() {
		errorutil.MustSucceed(rows.Close())
	}()

	var (
		id               int
		ts               int64
		category         Category
		rating           Rating
		contentTypeValue int
		contentBytes     []byte
	)

	result := []FetchedInsight{}

	for rows.Next() {
		err = rows.Scan(&id, &ts, &category, &rating, &contentTypeValue, &contentBytes)

		if err != nil {
			return []FetchedInsight{}, errorutil.Wrap(err)
		}

		contentType, err := ContentTypeForValue(contentTypeValue)

		if err != nil {
			return []FetchedInsight{}, errorutil.Wrap(err)
		}

		content, err := decodeByContentType(contentType, contentBytes)

		if err != nil {
			return []FetchedInsight{}, errorutil.Wrap(err)
		}

		result = append(result, &fetchedInsight{
			id:          id,
			time:        time.Unix(ts, 0).In(time.UTC),
			category:    category,
			rating:      rating,
			contentType: contentType,
			content:     content,
		})
	}

	if err := rows.Err(); err != nil {
		return []FetchedInsight{}, errorutil.Wrap(err)
	}

	return result, nil
}

type DBCreator struct {
	conn dbconn.RwConn
}

func NewCreator(conn dbconn.RwConn) (*DBCreator, error) {
	return &DBCreator{conn: conn}, nil
}

type InsightProperties struct {
	Time        time.Time `json:"time"`
	Category    Category  `json:"category"`
	Rating      Rating    `json:"rating"`
	ContentType string    `json:"content_type"`
	Content     Content   `json:"content"`
}

func (p InsightProperties) Title() notificationCore.ContentComponent {
	return p.Content.Title()
}

func (p InsightProperties) Description() notificationCore.ContentComponent {
	return p.Content.Description()
}

func (p InsightProperties) Metadata() notificationCore.ContentMetadata {
	return notificationCore.ContentMetadata{
		"category": p.Category,
		"priority": p.Rating,
	}
}

type Creator interface {
	GenerateInsight(context.Context, *sql.Tx, InsightProperties) error
}

func GenerateInsight(ctx context.Context, tx *sql.Tx, properties InsightProperties) (int64, error) {
	contentBytes, err := json.Marshal(properties.Content)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	log.Info().Msgf("Generating an insight with the content: %v", properties)

	contentTypeValue, err := ValueForContentType(properties.ContentType)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	result, err := tx.ExecContext(ctx,
		`insert into insights(time, category, rating, content_type, content) values(?, ?, ?, ?, ?)`,
		properties.Time.Unix(),
		properties.Category,
		properties.Rating,
		contentTypeValue,
		contentBytes)

	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, errorutil.Wrap(err)
	}

	if err := ArchiveInsightIfHistoricalImportIsRunning(ctx, tx, id, properties.Time); err != nil {
		return 0, errorutil.Wrap(err)
	}

	return id, nil
}
