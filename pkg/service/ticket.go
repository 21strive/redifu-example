package service

import (
	"database/sql"
	"errors"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"redifu-example/definition"
	"redifu-example/internal/fetcher"
	"redifu-example/internal/model"
	"redifu-example/internal/repository"
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

func (s *TicketService) Create(description string, accountUUID string, securityRisk int64) error {
	ticket := model.NewTicket()
	ticket.SetDescription(description)
	ticket.SetAccountUUID(accountUUID)
	ticket.SetSecurityRisk(securityRisk)

	return s.ticketRepository.Create(ticket)
}

func (s *TicketService) Find(ticketUUID string) (*model.Ticket, error) {
	ticket, errFind := s.ticketRepository.FindByUUID(ticketUUID)
	if errFind != nil {
		return nil, errFind
	}

	return ticket, nil
}

func (s *TicketService) UpdateDescription(ticketUUID string, description string) error {
	ticket, errFind := s.Find(ticketUUID)
	if errFind != nil {
		return errFind
	}

	ticket.SetDescription(description)
	return s.ticketRepository.Update(ticket)
}

func (s *TicketService) Delete(ticketUUID string) error {
	ticket, errFind := s.Find(ticketUUID)
	if errFind != nil {
		return errFind
	}

	return s.ticketRepository.Delete(ticket)
}

func (s *TicketService) ResolveTicket(ticketUUID string) error {
	ticket, errFind := s.Find(ticketUUID)
	if errFind != nil {
		return errFind
	}

	ticket.SetResolved()
	return s.ticketRepository.Update(ticket)
}

func (s *TicketService) GetTicket(randid string) (*model.Ticket, *model.Account, bool, error) {
	isBlank, err := s.ticketFetcher.IsBlank(randid)
	if err != nil {
		return nil, nil, false, err
	}
	if isBlank {
		return nil, nil, false, nil
	}

	ticket, errFetch := s.ticketFetcher.Fetch(randid)
	if errFetch != nil {
		return nil, nil, false, errFetch
	}

	accountFromCache, err := s.accountService.GetAccountByUUID(ticket.AccountUUID)
	if err != nil {
		if err == definition.NotFound {
			return ticket, nil, true, nil
		}
		return nil, nil, false, err
	}

	return ticket, accountFromCache, false, nil
}

func (s *TicketService) GetTickets(lastRandId []string) ([]*model.Ticket, string, string, bool, error) {
	tickets, validLastRandId, position, errFetch := s.ticketFetcher.FetchTimeline(lastRandId)
	if errFetch != nil {
		requiresSeed := false
		if errors.Is(errFetch, redifu.ResetPagination) {
			requiresSeed = true
		}
		return nil, validLastRandId, position, requiresSeed, errFetch
	}

	totalReceivedItems := int64(len(tickets))
	if totalReceivedItems < definition.ItemPerPage {
		seedRequired, errCheck := s.ticketFetcher.IsTimelineSeedingRequired(totalReceivedItems)
		if errCheck != nil {
			return nil, validLastRandId, position, false, errCheck
		}

		if seedRequired {
			return tickets, validLastRandId, position, true, nil
		}
	}

	return tickets, validLastRandId, position, false, nil
}

func (s *TicketService) GetTicketsByReporter(reporterUUID string) ([]*model.Ticket, bool, error) {
	tickets, errFetch := s.ticketFetcher.FetchSortedByReporter(reporterUUID)
	if errFetch != nil {
		return nil, false, errFetch
	}
	if len(tickets) == 0 {
		isSeedRequired, errCheck := s.ticketFetcher.IsSortedByReporterSeedingRequired(reporterUUID)
		if errCheck != nil {
			return nil, false, errCheck
		}
		if isSeedRequired {
			return nil, true, nil
		}
	}

	return tickets, false, nil
}

func (s *TicketService) GetTicketsBySecurityRisk(lastRandId []string) ([]*model.Ticket, string, string, bool, error) {
	tickets, validLastRandId, position, errFetch := s.ticketFetcher.FetchTimelineBySecurityRisk(lastRandId)
	if errFetch != nil {
		requiresSeed := false
		if errors.Is(errFetch, redifu.ResetPagination) {
			requiresSeed = true
		}
		return nil, validLastRandId, position, requiresSeed, errFetch
	}

	totalReceivedItems := int64(len(tickets))
	if totalReceivedItems < definition.ItemPerPage {
		seedRequired, errCheck := s.ticketFetcher.IsTimelineBySecurityRiskSeedingRequired(totalReceivedItems)
		if errCheck != nil {
			return nil, validLastRandId, position, false, errCheck
		}
		if seedRequired {
			return tickets, validLastRandId, position, true, nil
		}
	}

	return tickets, validLastRandId, position, false, nil
}

func (s *TicketService) GetTicketsByPage(page int64) ([]*model.Ticket, bool, error) {
	tickets, errFetch := s.ticketFetcher.FetchByPage(page)
	if errFetch != nil {
		return nil, false, errFetch
	}

	totalReceivedItems := int64(len(tickets))
	if totalReceivedItems == 0 {
		seedRequired, errCheck := s.ticketFetcher.IsTicketPageSeedRequired(page)
		if errCheck != nil {
			return nil, false, errCheck
		}

		return tickets, seedRequired, nil
	}

	return tickets, false, nil
}

func (s *TicketService) SeedTicket(randId string) error {
	errSeedTicket := s.ticketRepository.SeedTicket(randId)
	if errSeedTicket != nil {
		return errSeedTicket
	}

	ticketFromCache, errFetch := s.ticketFetcher.Fetch(randId)
	if errFetch != nil {
		return errFetch
	}

	errSeed := s.accountService.SeedAccountByUUID(ticketFromCache.AccountUUID)
	// allow system to seed target ticket although the reporter account is deleted/not exists
	if errSeed != nil && errSeed != definition.NotFound {
		return errSeed
	}

	return nil
}

func (s *TicketService) SeedTickets(subtraction int64, lastRandId string) error {
	return s.ticketRepository.SeedTickets(subtraction, lastRandId)
}

func (s *TicketService) SeedTicketsByAccount(reporterUUID string) error {
	return s.ticketRepository.SeedByAccount(reporterUUID)
}

func (s *TicketService) SeedTicketsBySecurityRisk(subtraction int64, lastRandId string) error {
	return s.ticketRepository.SeedTicketsBySecurityRisk(subtraction, lastRandId)
}

func (s *TicketService) SeedTicketsByPage(page int64) error {
	return s.ticketRepository.SeedPage(page)
}

func NewTicketService() *TicketService {
	return &TicketService{}
}
