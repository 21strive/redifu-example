package pools

import (
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"redifu-example/definition"
	"redifu-example/internal/model"
)

type FetcherPool struct {
	BaseTicket                 *redifu.Base[*model.Ticket]
	BaseAccount                *redifu.Base[*model.Account]
	Timeline                   *redifu.Timeline[*model.Ticket] // timeline
	TimelineByCategory         *redifu.Timeline[*model.Ticket] // timeline with param, query & relation
	TimelineSortBySecurityRisk *redifu.Timeline[*model.Ticket] // timeline sort by custom parameter
	SortedByAccount            *redifu.Sorted[*model.Ticket]
	Page                       *redifu.Page[*model.Ticket]
	TimeSeries                 *redifu.TimeSeries[*model.Ticket]
}

func NewFetcherPool(redisClient redis.UniversalClient) *FetcherPool {
	base := redifu.NewBase[*model.Ticket](redisClient, "ticket:%s", definition.BaseTTL)
	baseAccount := redifu.NewBase[*model.Account](redisClient, "account:%s", definition.BaseTTL)
	baseCategory := redifu.NewBase[*model.Category](redisClient, "category:%s", definition.BaseTTL)

	accountRelation := redifu.NewRelation[*model.Account](baseAccount, redifu.TypeOf[model.Ticket]())
	categoryRelation := redifu.NewRelation[*model.Category](baseCategory, redifu.TypeOf[model.Ticket]())

	timeline := redifu.NewTimeline[*model.Ticket](redisClient, base, "ticket-timeline", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	timeline.AddRelation("account", accountRelation)
	timeline.AddRelation("category", categoryRelation)

	timelineByCategory := redifu.NewTimeline[*model.Ticket](redisClient, base, "ticket-timeline:category:%s", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	timelineByCategory.AddRelation("account", accountRelation)
	timelineByCategory.AddRelation("category", categoryRelation)

	timelineSortBySecurityRisk := redifu.NewTimeline[*model.Ticket](redisClient, base, "ticket-timeline-by-security", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	timelineSortBySecurityRisk.AddRelation("account", accountRelation)
	timelineSortBySecurityRisk.AddRelation("category", categoryRelation)
	timelineSortBySecurityRisk.SetSortingReference("SecurityRisk")

	sortedByAccount := redifu.NewSorted[*model.Ticket](redisClient, base, "ticket-sorted-by-account", definition.SortedSetTTL)
	sortedByAccount.AddRelation("account", accountRelation)

	page := redifu.NewPage[*model.Ticket](redisClient, base, "ticket-page", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	page.AddRelation("account", accountRelation)

	timeSeries := redifu.NewTimeSeries[*model.Ticket](redisClient, base, "ticket-time-series", definition.SortedSetTTL)
	timeSeries.AddRelation("account", accountRelation)

	return &FetcherPool{
		BaseTicket:                 base,
		BaseAccount:                baseAccount,
		Timeline:                   timeline,
		TimelineByCategory:         timelineByCategory,
		TimelineSortBySecurityRisk: timelineSortBySecurityRisk,
		SortedByAccount:            sortedByAccount,
		Page:                       page,
		TimeSeries:                 timeSeries,
	}
}
