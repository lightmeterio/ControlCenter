// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
	case ActiveCategory:
		return translator.I18n("active")
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
	LocalCategory       Category = 1 // insights generated from logs - high bounce/deferred, mail activity, rbl checks...
	NewsCategory        Category = 2 // insights showing lightmeter.io RSS items
	ComparativeCategory Category = 3 // not used yet
	IntelCategory       Category = 4 // sent by Network Intelligence central server
	ArchivedCategory    Category = 5 // meta-category, not stored in database
	ActiveCategory      Category = 6 // same, opposite to archived
)

func (c Category) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

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
	case "active":
		return ActiveCategory
	default:
		return NoCategory
	}
}

func (c *Category) UnmarshalJSON(b []byte) error {
	var s string

	if err := json.Unmarshal(b, &s); err != nil {
		return errorutil.Wrap(err)
	}

	*c = BuildCategoryByName(s)

	return nil
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

func (r Rating) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

var ErrInvalidRating = errors.New(`Invalid Rating`)

func (r *Rating) UnmarshalJSON(b []byte) error {
	var s string

	if err := json.Unmarshal(b, &s); err != nil {
		return errorutil.Wrap(err)
	}

	switch s {
	case "bad":
		*r = BadRating
		return nil
	case "ok":
		*r = OkRating
		return nil
	case "good":
		*r = GoodRating
		return nil
	case "unrated":
		*r = Unrated
		return nil
	default:
		return ErrInvalidRating
	}
}

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
	UserRating() *int
	UserRatingOld() bool
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
	Clock      interface{}
}

type Fetcher interface {
	FetchInsights(context.Context, FetchOptions, timeutil.Clock) ([]FetchedInsight, error)
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
	// active_category is the one stored in the `insights` table. It's immutable.
	// status_category is the one the user sees, and might change over time (like from active to archived).
	return fmt.Sprintf(`
	with
	user_ratings (insight_type, rating, timestamp) as (
		select *
		from insights_user_ratings iur
		where not exists (
			select *
			from insights_user_ratings iur2
			where iur2.insight_type = iur.insight_type
			  and iur2.timestamp > iur.timestamp  -- only the last user rating per insight_type
		)
		group by insight_type  -- edge case, several ratings on the same second
	),
	insights_with_status_rating(id, time, actual_category, status_category, rating, content_type, content, user_rating, user_rating_ts) as (
		select
			insights.rowid, insights.time, insights.category, ifnull(insights_status.status, %d), insights.rating, insights.content_type, insights.content, user_ratings.rating, iif(user_ratings.timestamp is not null, user_ratings.timestamp, 0)
		from
			insights
		left join insights_status on insights.rowid = insights_status.insight_id
		left join user_ratings on insights.content_type = user_ratings.insight_type
	)
	select
		id, time, iif(status_category == %d, status_category, actual_category) as computed_category, rating, content_type, content, user_rating, user_rating_ts
	from
		insights_with_status_rating
	where %s
	order by %s, id
	limit @limit
	`, int(ActiveCategory), int(ArchivedCategory), where, order)
}

var (
	noFilterSqlWhereClause = `time between @start and @end`

	// TODO: we should analyze and simplify and optmize this condition, as it's unreadable!
	filterByCategorySqlWhereClause = fmt.Sprintf(`time between @start and @end and ((@category in (%d, %d) and status_category = @category) or (@category not in (%d, %d) and @category = actual_category and status_category != %d))`, int(ActiveCategory), int(ArchivedCategory), int(ActiveCategory), int(ArchivedCategory), int(ArchivedCategory))
)

func noFilterParamBuilder(o FetchOptions) []interface{} {
	return []interface{}{
		sql.Named("start", o.Interval.From.Unix()),
		sql.Named("end", o.Interval.To.Unix()),
		sql.Named("limit", buildLimitForFetchOptions(o)),
	}
}

func filterByCategoryParamBuilder(o FetchOptions) []interface{} {
	return []interface{}{
		sql.Named("start", o.Interval.From.Unix()),
		sql.Named("end", o.Interval.To.Unix()),
		sql.Named("category", o.Category),
		sql.Named("limit", buildLimitForFetchOptions(o)),
	}
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

			c.SetStmt(key, q)

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
				return buildQuery(key, buildSelectStmt(noFilterSqlWhereClause, `time desc`))
			},
			paramBuilder: noFilterParamBuilder,
		},
		{
			key: queryKey{order: OrderByCreationDesc, filter: FilterByCategory},
			value: func(key queryKey) error {
				return buildQuery(key, buildSelectStmt(filterByCategorySqlWhereClause, `time desc`))
			},
			paramBuilder: filterByCategoryParamBuilder,
		},
		{
			key: queryKey{order: OrderByCreationAsc, filter: FilterByCategory},
			value: func(key queryKey) error {
				return buildQuery(key, buildSelectStmt(filterByCategorySqlWhereClause, `time asc`))
			},
			paramBuilder: filterByCategoryParamBuilder,
		},
		{
			key: queryKey{order: OrderByCreationAsc, filter: NoFetchFilter},
			value: func(key queryKey) error {
				return buildQuery(key, buildSelectStmt(noFilterSqlWhereClause, `time asc`))
			},
			paramBuilder: noFilterParamBuilder,
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
	id            int
	time          time.Time
	rating        Rating
	category      Category
	contentType   string
	content       Content
	userRating    *int
	userRatingOld bool
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

func (f *fetchedInsight) UserRating() *int {
	return f.userRating
}

func (f *fetchedInsight) UserRatingOld() bool {
	return f.userRatingOld
}

// rowserrcheck is not able to notice that query.Err() is called and emits a false positive warning
//nolint:rowserrcheck
func (f *fetcher) FetchInsights(ctx context.Context, options FetchOptions, clock timeutil.Clock) ([]FetchedInsight, error) {
	conn, release, err := f.pool.AcquireContext(ctx)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer release()

	key := queryKey{order: options.OrderBy, filter: options.FilterBy}

	//nolint:sqlclosecheck
	stmt := conn.GetStmt(key)
	query := f.queries[key]

	rows, err := stmt.QueryContext(ctx, query.p(options)...)

	if err != nil {
		return nil, errorutil.Wrap(err)
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
		userRating       *int
		userRatingTs     int64
		userRatingOld    bool
	)

	result := []FetchedInsight{}

	for rows.Next() {
		err = rows.Scan(&id, &ts, &category, &rating, &contentTypeValue, &contentBytes, &userRating, &userRatingTs)

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

		// rating is old if more than two weeks old (or not rated yet, epoch is more than two weeks old)
		userRatingOld = insightUserRatingIsOld(time.Unix(userRatingTs, 0), clock)

		result = append(result, &fetchedInsight{
			id:            id,
			time:          time.Unix(ts, 0).In(time.UTC),
			category:      category,
			rating:        rating,
			contentType:   contentType,
			content:       content,
			userRating:    userRating,
			userRatingOld: userRatingOld,
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
	Time           time.Time `json:"time"`
	Category       Category  `json:"category"`
	Rating         Rating    `json:"rating"`
	ContentType    string    `json:"content_type"`
	Content        Content   `json:"content"`
	MustBeNotified bool      `json:"-"`
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
