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
}

func (s *TicketService) InitRepository(db *sql.DB, redisClient redis.UniversalClient) {
	ticketRepository := repository.NewTicketRepository(db, redisClient)
	s.ticketRepository = ticketRepository
}

func (s *TicketService) InitFetcher(redisClient redis.UniversalClient) {
	ticketFetcher := fetcher.NewTicketFetcher(redisClient)
	s.ticketFetcher = ticketFetcher
}

func (s *TicketService) Create(description string, reporterUUID string) error {
	ticket := model.NewTicket()
	ticket.SetDescription(description)
	ticket.SetReporterUUID(reporterUUID)

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

func (s *TicketService) GetTicket(randid string) (*model.Ticket, bool, error) {
	isBlank, err := s.ticketFetcher.IsBlank(randid)
	if err != nil {
		return nil, false, err
	}
	if isBlank {
		return nil, true, nil
	}

	ticket, errFetch := s.ticketFetcher.Fetch(randid)
	if errFetch != nil {
		return nil, false, errFetch
	}

	return ticket, false, nil
}

func (s *TicketService) GetTickets(lastRandId []string) ([]model.Ticket, string, string, bool, error) {
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

func (s *TicketService) GetTicketsByReporter(reporterUUID string) ([]model.Ticket, bool, error) {
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

func NewTicketService() *TicketService {
	return &TicketService{}
}
