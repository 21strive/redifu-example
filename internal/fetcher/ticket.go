package fetcher

import (
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"redifu-example/definition"
	"redifu-example/internal/model"
)

type TicketFetcher struct {
	base               *redifu.Base[*model.Ticket]
	timeline           *redifu.Timeline[*model.Ticket]
	timelineByReporter *redifu.Timeline[*model.Ticket]
	sorted             *redifu.Sorted[*model.Ticket]
	sortedByReporter   *redifu.Sorted[*model.Ticket]
}

func (t *TicketFetcher) Init(base *redifu.Base[*model.Ticket], timeline *redifu.Timeline[*model.Ticket], sortedByAccount *redifu.Sorted[*model.Ticket]) {
	t.base = base
	t.timeline = timeline
	t.sorted = sortedByAccount
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

func (t *TicketFetcher) FetchTimeline(lastRandId []string) ([]*model.Ticket, string, string, error) {
	return t.timeline.Fetch(nil, lastRandId, nil, nil)
}

func (t *TicketFetcher) IsTimelineSeedingRequired(totalReceivedItem int64) (bool, error) {
	return t.timeline.RequiresSeeding(nil, totalReceivedItem)
}

func (t *TicketFetcher) FetchSortedByReporter(reporterUUID string) ([]*model.Ticket, error) {
	return t.sorted.Fetch([]string{reporterUUID}, redifu.Descending, nil, nil)
}

func (t *TicketFetcher) IsSortedByReporterSeedingRequired(reporterUUID string) (bool, error) {
	return t.sorted.RequiresSeeding([]string{reporterUUID})
}

func NewTicketFetcher(redisClient redis.UniversalClient) *TicketFetcher {
	base := redifu.NewBase[*model.Ticket](redisClient, "ticket:%s", definition.BaseTTL)
	baseAccount := redifu.NewBase[*model.Account](redisClient, "account:%s", definition.BaseTTL)
	accountRelation := redifu.NewRelation[*model.Account](baseAccount, "Account", "AccountRandId")

	timeline := redifu.NewTimeline[*model.Ticket](redisClient, base, "ticket-timeline", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	timeline.AddRelation("account", accountRelation)

	sortedByAccount := redifu.NewSorted[*model.Ticket](redisClient, base, "ticket-sorted-by-account", definition.SortedSetTTL)
	sortedByAccount.AddRelation("account", accountRelation)

	ticketFetcher := &TicketFetcher{}
	ticketFetcher.Init(base, timeline, sortedByAccount)
	return ticketFetcher
}
