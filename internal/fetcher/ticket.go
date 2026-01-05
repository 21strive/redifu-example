package fetcher

import (
	"context"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"redifu-example/definition"
	"redifu-example/internal/model"
	"time"
)

type TicketFetcher struct {
	base                   *redifu.Base[*model.Ticket]
	timeline               *redifu.Timeline[*model.Ticket]
	timelineBySecurityRisk *redifu.Timeline[*model.Ticket]
	sortedByAccount        *redifu.Sorted[*model.Ticket]
	page                   *redifu.Page[*model.Ticket]
	timeSeries             *redifu.TimeSeries[*model.Ticket]
}

func (t *TicketFetcher) Init(
	base *redifu.Base[*model.Ticket],
	timeline *redifu.Timeline[*model.Ticket],
	timelineBySecurityRisk *redifu.Timeline[*model.Ticket],
	sortedByAccount *redifu.Sorted[*model.Ticket],
	page *redifu.Page[*model.Ticket],
	timeSeries *redifu.TimeSeries[*model.Ticket],
) {
	t.base = base
	t.timeline = timeline
	t.timelineBySecurityRisk = timelineBySecurityRisk
	t.sortedByAccount = sortedByAccount
	t.page = page
	t.timeSeries = timeSeries
}

func (t *TicketFetcher) Fetch(randid string) (*model.Ticket, error) {
	ticket, err := t.base.Get(randid)
	if err != nil {
		return nil, err
	}

	return ticket, nil
}

func (t *TicketFetcher) IsBlank(randid string) (bool, error) {
	return t.base.IsBlank(randid)
}

func (t *TicketFetcher) FetchTimeline(ctx context.Context, lastRandId []string) ([]*model.Ticket, string, string, error) {
	return t.timeline.Fetch(lastRandId).Exec(ctx)
}

func (t *TicketFetcher) IsTimelineSeedingRequired(ctx context.Context, totalReceivedItem int64) (bool, error) {
	return t.timeline.RequiresSeeding(totalReceivedItem).Exec(ctx)
}

func (t *TicketFetcher) FetchTimelineBySecurityRisk(ctx context.Context, lastRandId []string) ([]*model.Ticket, string, string, error) {
	return t.timelineBySecurityRisk.Fetch(lastRandId).Exec(ctx)
}

func (t *TicketFetcher) IsTimelineBySecurityRiskSeedingRequired(ctx context.Context, totalReceivedItem int64) (bool, error) {
	return t.timelineBySecurityRisk.RequiresSeeding(totalReceivedItem).Exec(ctx)
}

func (t *TicketFetcher) FetchSortedByReporter(reporterUUID string) ([]*model.Ticket, error) {
	return t.sortedByAccount.Fetch([]string{reporterUUID}, redifu.Descending, nil, nil)
}

func (t *TicketFetcher) IsSortedByReporterSeedingRequired(reporterUUID string) (bool, error) {
	return t.sortedByAccount.RequiresSeeding([]string{reporterUUID})
}

func (t *TicketFetcher) FetchByPage(page int64) ([]*model.Ticket, error) {
	return t.page.Fetch(page).Exec()
}

func (t *TicketFetcher) IsTicketPageSeedRequired(page int64) (bool, error) {
	return t.page.RequiresSeeding(page).Exec()
}

func (t *TicketFetcher) FetchByRange(lowerbound time.Time, upperbound time.Time) ([]*model.Ticket, bool, error) {
	return t.timeSeries.Fetch(lowerbound, upperbound, nil, nil, nil)
}

func NewTicketFetcher(redisClient redis.UniversalClient) *TicketFetcher {
	base := redifu.NewBase[*model.Ticket](redisClient, "ticket:%s", definition.BaseTTL)
	baseAccount := redifu.NewBase[*model.Account](redisClient, "account:%s", definition.BaseTTL)
	accountRelation := redifu.NewRelation[*model.Account](baseAccount, "Account", "AccountRandId")

	timeline := redifu.NewTimeline[*model.Ticket](
		redisClient,
		base,
		"ticket-timeline",
		definition.ItemPerPage,
		redifu.Descending,
		definition.SortedSetTTL)
	timeline.AddRelation("account", accountRelation)

	timelineBySecurityRisk := redifu.NewTimeline[*model.Ticket](
		redisClient,
		base,
		"ticket-timeline",
		definition.ItemPerPage,
		redifu.Descending,
		definition.SortedSetTTL)
	timelineBySecurityRisk.AddRelation("account", accountRelation)
	timelineBySecurityRisk.SetSortingReference("SecurityRisk")

	sortedByAccount := redifu.NewSorted[*model.Ticket](
		redisClient,
		base,
		"ticket-sorted-by-account",
		definition.SortedSetTTL)
	sortedByAccount.AddRelation("account", accountRelation)

	page := redifu.NewPage[*model.Ticket](
		redisClient,
		base,
		"ticket-page",
		definition.ItemPerPage,
		redifu.Descending,
		definition.SortedSetTTL)
	page.AddRelation("account", accountRelation)

	timeSeries := redifu.NewTimeSeries[*model.Ticket](
		redisClient,
		base,
		"ticket-time-series",
		definition.SortedSetTTL)

	ticketFetcher := &TicketFetcher{}
	ticketFetcher.Init(base, timeline, timelineBySecurityRisk, sortedByAccount, page, timeSeries)
	return ticketFetcher
}
