package service

import (
	"context"
	"database/sql"
	"errors"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"redifu-example/definition"
	"redifu-example/internal/fetcher"
	"redifu-example/internal/model"
	"redifu-example/internal/repository"
	"time"
)

type TicketService struct {
	ticketRepository *repository.TicketRepository
	ticketFetcher    *fetcher.TicketFetcher
	accountService   *AccountService
}

func (s *TicketService) InitRepository(db *sql.DB, redisClient redis.UniversalClient, accountService *AccountService) {
	ticketRepository := repository.NewTicketRepository(db, redisClient)
	s.ticketRepository = ticketRepository
	s.accountService = accountService
}

func (s *TicketService) InitFetcher(redisClient redis.UniversalClient) {
	ticketFetcher := fetcher.NewTicketFetcher(redisClient)
	s.ticketFetcher = ticketFetcher
}

func (s *TicketService) Create(ctx context.Context, description string, accountUUID string, securityRisk int64) error {
	ticket := model.NewTicket()
	ticket.SetDescription(description)
	ticket.SetAccountUUID(accountUUID)
	ticket.SetSecurityRisk(securityRisk)

	return s.ticketRepository.Create(ctx, ticket)
}

func (s *TicketService) Find(ctx context.Context, ticketUUID string) (*model.Ticket, error) {
	ticket, errFind := s.ticketRepository.FindByUUID(ctx, ticketUUID)
	if errFind != nil {
		return nil, errFind
	}

	return ticket, nil
}

func (s *TicketService) UpdateDescription(ctx context.Context, ticketUUID string, description string) error {
	ticket, errFind := s.Find(ctx, ticketUUID)
	if errFind != nil {
		return errFind
	}

	ticket.SetDescription(description)
	return s.ticketRepository.Update(ctx, ticket)
}

func (s *TicketService) Delete(ctx context.Context, ticketUUID string) error {
	ticket, errFind := s.Find(ctx, ticketUUID)
	if errFind != nil {
		return errFind
	}

	return s.ticketRepository.Delete(ctx, ticket)
}

func (s *TicketService) ResolveTicket(ctx context.Context, ticketUUID string) error {
	ticket, errFind := s.Find(ctx, ticketUUID)
	if errFind != nil {
		return errFind
	}

	ticket.SetResolved()
	return s.ticketRepository.Update(ctx, ticket)
}

func (s *TicketService) GetTicket(ctx context.Context, randid string) (*model.Ticket, *model.Account, bool, error) {
	isBlank, err := s.ticketFetcher.IsBlank(ctx, randid)
	if err != nil {
		return nil, nil, false, err
	}
	if isBlank {
		return nil, nil, false, nil
	}

	ticket, errFetch := s.ticketFetcher.Fetch(ctx, randid)
	if errFetch != nil {
		return nil, nil, false, errFetch
	}

	accountFromCache, err := s.accountService.GetAccountByUUID(ctx, ticket.AccountUUID)
	if err != nil {
		if err == definition.NotFound {
			return ticket, nil, true, nil
		}
		return nil, nil, false, err
	}

	return ticket, accountFromCache, false, nil
}

func (s *TicketService) GetTickets(ctx context.Context, lastRandId []string) ([]*model.Ticket, string, string, bool, error) {
	tickets, validLastRandId, position, errFetch := s.ticketFetcher.FetchTimeline(ctx, lastRandId)
	if errFetch != nil {
		requiresSeed := false
		if errors.Is(errFetch, redifu.ResetPagination) {
			requiresSeed = true
		}
		return nil, validLastRandId, position, requiresSeed, errFetch
	}

	totalReceivedItems := int64(len(tickets))
	if totalReceivedItems < definition.ItemPerPage {
		seedRequired, errCheck := s.ticketFetcher.IsTimelineSeedingRequired(ctx, totalReceivedItems)
		if errCheck != nil {
			return nil, validLastRandId, position, false, errCheck
		}

		if seedRequired {
			return tickets, validLastRandId, position, true, nil
		}
	}

	return tickets, validLastRandId, position, false, nil
}

func (s *TicketService) GetTicketsByReporter(ctx context.Context, reporterUUID string) ([]*model.Ticket, bool, error) {
	tickets, errFetch := s.ticketFetcher.FetchSortedByReporter(ctx, reporterUUID)
	if errFetch != nil {
		return nil, false, errFetch
	}
	if len(tickets) == 0 {
		isSeedRequired, errCheck := s.ticketFetcher.IsSortedByReporterSeedingRequired(ctx, reporterUUID)
		if errCheck != nil {
			return nil, false, errCheck
		}
		if isSeedRequired {
			return nil, true, nil
		}
	}

	return tickets, false, nil
}

func (s *TicketService) GetTicketsBySecurityRisk(ctx context.Context, lastRandId []string) ([]*model.Ticket, string, string, bool, error) {
	tickets, validLastRandId, position, errFetch := s.ticketFetcher.FetchTimelineBySecurityRisk(ctx, lastRandId)
	if errFetch != nil {
		requiresSeed := false
		if errors.Is(errFetch, redifu.ResetPagination) {
			requiresSeed = true
		}
		return nil, validLastRandId, position, requiresSeed, errFetch
	}

	totalReceivedItems := int64(len(tickets))
	if totalReceivedItems < definition.ItemPerPage {
		seedRequired, errCheck := s.ticketFetcher.IsTimelineBySecurityRiskSeedingRequired(ctx, totalReceivedItems)
		if errCheck != nil {
			return nil, validLastRandId, position, false, errCheck
		}
		if seedRequired {
			return tickets, validLastRandId, position, true, nil
		}
	}

	return tickets, validLastRandId, position, false, nil
}

func (s *TicketService) GetTicketsByPage(ctx context.Context, page int64) ([]*model.Ticket, bool, error) {
	tickets, errFetch := s.ticketFetcher.FetchByPage(ctx, page)
	if errFetch != nil {
		return nil, false, errFetch
	}

	totalReceivedItems := int64(len(tickets))
	if totalReceivedItems == 0 {
		seedRequired, errCheck := s.ticketFetcher.IsTicketPageSeedRequired(ctx, page)
		if errCheck != nil {
			return nil, false, errCheck
		}

		return tickets, seedRequired, nil
	}

	return tickets, false, nil
}

func (s *TicketService) GetTicketsByDate(ctx context.Context, lowerbound time.Time, upperbound time.Time) ([]*model.Ticket, bool, error) {
	return s.ticketFetcher.FetchByRange(ctx, lowerbound, upperbound)
}

func (s *TicketService) SeedTicket(ctx context.Context, randId string) error {
	errSeedTicket := s.ticketRepository.SeedTicket(ctx, randId)
	if errSeedTicket != nil {
		return errSeedTicket
	}

	ticketFromCache, errFetch := s.ticketFetcher.Fetch(ctx, randId)
	if errFetch != nil {
		return errFetch
	}

	errSeed := s.accountService.SeedAccountByUUID(ctx, ticketFromCache.AccountUUID)
	// allow system to seed target ticket although the reporter account is deleted/not exists
	if errSeed != nil && errSeed != definition.NotFound {
		return errSeed
	}

	return nil
}

func (s *TicketService) SeedTickets(ctx context.Context, subtraction int64, lastRandId string) error {
	return s.ticketRepository.SeedTickets(ctx, subtraction, lastRandId)
}

func (s *TicketService) SeedTicketsByAccount(ctx context.Context, reporterUUID string) error {
	return s.ticketRepository.SeedByAccount(ctx, reporterUUID)
}

func (s *TicketService) SeedTicketsBySecurityRisk(ctx context.Context, subtraction int64, lastRandId string) error {
	return s.ticketRepository.SeedTicketsBySecurityRisk(ctx, subtraction, lastRandId)
}

func (s *TicketService) SeedTicketsByPage(ctx context.Context, page int64) error {
	return s.ticketRepository.SeedPage(ctx, page)
}

func (s *TicketService) SeedTicketsByDate(ctx context.Context, lowerbound time.Time, upperbound time.Time) error {
	return s.ticketRepository.SeedByDate(ctx, lowerbound, upperbound)
}

func NewTicketService() *TicketService {
	return &TicketService{}
}
