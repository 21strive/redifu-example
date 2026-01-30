package fetcher

import (
	"context"
	"github.com/21strive/redifu"
	"redifu-example/internal/model"
	"redifu-example/internal/pools"
	"time"
)

type TicketFetcher struct {
	base                   *redifu.Base[*model.Ticket]
	timeline               *redifu.Timeline[*model.Ticket]
	timelineByCategory     *redifu.Timeline[*model.Ticket]
	timelineBySecurityRisk *redifu.Timeline[*model.Ticket]
	sortedByAccount        *redifu.Sorted[*model.Ticket]
	page                   *redifu.Page[*model.Ticket]
	timeSeries             *redifu.TimeSeries[*model.Ticket]
}

func (t *TicketFetcher) Init(
	base *redifu.Base[*model.Ticket],
	timeline *redifu.Timeline[*model.Ticket],
	timelineByCategory *redifu.Timeline[*model.Ticket],
	timelineBySecurityRisk *redifu.Timeline[*model.Ticket],
	sortedByAccount *redifu.Sorted[*model.Ticket],
	page *redifu.Page[*model.Ticket],
	timeSeries *redifu.TimeSeries[*model.Ticket],
) {
	t.base = base
	t.timeline = timeline
	t.timelineByCategory = timelineByCategory
	t.timelineBySecurityRisk = timelineBySecurityRisk
	t.sortedByAccount = sortedByAccount
	t.page = page
	t.timeSeries = timeSeries
}

func (t *TicketFetcher) Fetch(ctx context.Context, randid string) (*model.Ticket, error) {
	ticket, err := t.base.Get(ctx, randid)
	if err != nil {
		return nil, err
	}

	return ticket, nil
}

func (t *TicketFetcher) IsBlank(ctx context.Context, randid string) (bool, error) {
	return t.base.IsMissing(ctx, randid)
}

func (t *TicketFetcher) FetchTimeline(ctx context.Context, lastRandId []string) *redifu.FetchOutput[*model.Ticket] {
	return t.timeline.Fetch(lastRandId).Exec(ctx)
}

func (t *TicketFetcher) IsTimelineSeedingRequired(ctx context.Context, totalReceivedItem int64) (bool, error) {
	return t.timeline.RequiresSeeding(ctx, totalReceivedItem)
}

func (t *TicketFetcher) FetchTimelineByCategory(ctx context.Context, categoryRandId string, lastRandId []string) *redifu.FetchOutput[*model.Ticket] {
	return t.timelineByCategory.Fetch(lastRandId).WithParams(categoryRandId).Exec(ctx)
}

func (t *TicketFetcher) IsTimelineByCategorySeedingRequired(ctx context.Context, categoryRandId string, totalReceivedItem int64) (bool, error) {
	return t.timelineByCategory.RequiresSeeding(ctx, totalReceivedItem, categoryRandId)
}

func (t *TicketFetcher) FetchTimelineBySecurityRisk(ctx context.Context, lastRandId []string) *redifu.FetchOutput[*model.Ticket] {
	return t.timelineBySecurityRisk.Fetch(lastRandId).Exec(ctx)
}

func (t *TicketFetcher) IsTimelineBySecurityRiskSeedingRequired(ctx context.Context, totalReceivedItem int64) (bool, error) {
	return t.timelineBySecurityRisk.RequiresSeeding(ctx, totalReceivedItem)
}

func (t *TicketFetcher) FetchSortedByReporter(ctx context.Context, reporterUUID string) ([]*model.Ticket, error) {
	return t.sortedByAccount.Fetch(redifu.Descending).WithParams(reporterUUID).Exec(ctx)
}

func (t *TicketFetcher) IsSortedByReporterSeedingRequired(ctx context.Context, reporterUUID string) (bool, error) {
	return t.sortedByAccount.RequiresSeeding(ctx, reporterUUID)
}

func (t *TicketFetcher) FetchByPage(ctx context.Context, page int64) ([]*model.Ticket, error) {
	return t.page.Fetch(page).Exec(ctx)
}

func (t *TicketFetcher) IsTicketPageSeedRequired(ctx context.Context, page int64) (bool, error) {
	return t.page.RequiresSeeding(ctx, page)
}

func (t *TicketFetcher) FetchByRange(ctx context.Context, lowerbound time.Time, upperbound time.Time) ([]*model.Ticket, bool, error) {
	return t.timeSeries.Fetch(lowerbound, upperbound).Exec(ctx)
}

func NewTicketFetcher(fetcherPool *pools.FetcherPool) *TicketFetcher {
	ticketFetcher := &TicketFetcher{}
	ticketFetcher.Init(
		fetcherPool.BaseTicket,
		fetcherPool.Timeline,
		fetcherPool.TimelineByCategory,
		fetcherPool.TimelineSortBySecurityRisk,
		fetcherPool.SortedByAccount,
		fetcherPool.Page,
		fetcherPool.TimeSeries)
	return ticketFetcher
}
