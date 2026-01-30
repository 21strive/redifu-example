package pools

import (
	"database/sql"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"redifu-example/internal/model"
)

type SeederPool struct {
	TimelineSeeder                   *redifu.TimelineSeeder[*model.Ticket]
	TimelineSortBySecurityRiskSeeder *redifu.TimelineSeeder[*model.Ticket]
	TimelineByCategorySeeder         *redifu.TimelineSeeder[*model.Ticket]
	SortedByAccountSeeder            *redifu.SortedSeeder[*model.Ticket]
	PageSeeder                       *redifu.PageSeeder[*model.Ticket]
	TimeSeriesSeeder                 *redifu.TimeSeriesSeeder[*model.Ticket]
}

func (s *SeederPool) InitTicketSeeder(redisClient redis.UniversalClient, readDB *sql.DB, baseTicket *redifu.Base[*model.Ticket], timelineTicket *redifu.Timeline[*model.Ticket]) {
	s.TimelineSeeder = redifu.NewTimelineSeeder[*model.Ticket](redisClient, readDB, baseTicket, timelineTicket)
}

func (s *SeederPool) InitTicketBySecurityRiskSeeder(redisClient redis.UniversalClient, readDB *sql.DB, baseTicket *redifu.Base[*model.Ticket], timelineTicket *redifu.Timeline[*model.Ticket]) {
	s.TimelineSortBySecurityRiskSeeder = redifu.NewTimelineSeeder[*model.Ticket](redisClient, readDB, baseTicket, timelineTicket)
}

func (s *SeederPool) InitTicketByCategorySeeder(redisClient redis.UniversalClient, readDB *sql.DB, baseTicket *redifu.Base[*model.Ticket], timelineTicket *redifu.Timeline[*model.Ticket]) {
	s.TimelineByCategorySeeder = redifu.NewTimelineSeeder[*model.Ticket](redisClient, readDB, baseTicket, timelineTicket)
}

func (s *SeederPool) InitTicketByAccountSeeder(redisClient redis.UniversalClient, readDB *sql.DB, baseTicket *redifu.Base[*model.Ticket], sortedTicket *redifu.Sorted[*model.Ticket]) {
	s.SortedByAccountSeeder = redifu.NewSortedSeeder[*model.Ticket](redisClient, readDB, baseTicket, sortedTicket)
}

func (s *SeederPool) InitTicketPageSeeder(redisClient redis.UniversalClient, readDB *sql.DB, baseTicket *redifu.Base[*model.Ticket], pageTicket *redifu.Page[*model.Ticket]) {
	s.PageSeeder = redifu.NewPageSeeder[*model.Ticket](redisClient, readDB, baseTicket, pageTicket)
}

func (s *SeederPool) InitTicketTimeSeriesSeeder(redisClient redis.UniversalClient, readDB *sql.DB, baseTicket *redifu.Base[*model.Ticket], timeSeriesTicket *redifu.TimeSeries[*model.Ticket]) {
	s.TimeSeriesSeeder = redifu.NewTimeSeriesSeeder[*model.Ticket](redisClient, readDB, baseTicket, timeSeriesTicket)
}

func NewSeederPool() *SeederPool {
	return &SeederPool{}
}
