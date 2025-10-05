package main

import (
	"github.com/gofiber/fiber/v2"
	"redifu-example/definition"
	"redifu-example/lib"
)

type CommonError struct {
	Status int
	Code   string
	Error  error
}

func CreateTicket(request CreateTicketRequest, ticketRepository *lib.TicketRepository) *CommonError {
	ticket := lib.NewTicket()
	ticket.SetDescription(request.Description)

	errCreate := ticketRepository.Create(ticket)
	if errCreate != nil {
		return &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errCreate,
		}
	}

	return nil
}

func UpdateDescription(request UpdateTicketDescriptionRequest, ticketRepository *lib.TicketRepository) *CommonError {
	ticket, errFind := ticketRepository.FindByUUID(request.TicketUUID)
	if errFind != nil {
		return &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errFind,
		}
	}
	if ticket == nil {
		return &CommonError{
			Status: fiber.StatusNotFound,
			Code:   "T404",
			Error:  nil,
		}
	}

	ticket.SetDescription(request.Description)
	errUpdate := ticketRepository.Update(ticket)
	if errUpdate != nil {
		return &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errUpdate,
		}
	}

	return nil
}

func DeleteTicket(request UpdateTicketDescriptionRequest, ticketRepository *lib.TicketRepository) *CommonError {
	ticket, errFind := ticketRepository.FindByUUID(request.TicketUUID)
	if errFind != nil {
		return &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errFind,
		}
	}
	if ticket == nil {
		return &CommonError{
			Status: fiber.StatusNotFound,
			Code:   "T404",
			Error:  nil,
		}
	}

	errDelete := ticketRepository.Delete(ticket)
	if errDelete != nil {
		return &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errDelete,
		}
	}

	return nil
}

func ResolveTicket(ticketUUID string, ticketRepository *lib.TicketRepository) *CommonError {
	ticket, errFind := ticketRepository.FindByUUID(ticketUUID)
	if errFind != nil {
		return &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errFind,
		}
	}
	if ticket == nil {
		return &CommonError{
			Status: fiber.StatusNotFound,
			Code:   "T404",
			Error:  nil,
		}
	}

	ticket.SetResolved()
	errUpdate := ticketRepository.Update(ticket)
	if errUpdate != nil {
		return &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errUpdate,
		}
	}

	return nil
}

func Fetch(randid string, ticketFetcher *lib.TicketFetcher) (*lib.Ticket, bool, *CommonError) {
	isBlank, err := ticketFetcher.IsBlank(randid)
	if err != nil {
		return nil, false, &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  err,
		}
	}
	if isBlank {
		return nil, true, nil
	}

	ticket, errFetch := ticketFetcher.Fetch(randid)
	if errFetch != nil {
		return nil, false, &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errFetch,
		}
	}

	return ticket, false, nil
}

func FetchTimeline(lastRandId []string, ticketFetcher *lib.TicketFetcher) ([]lib.Ticket, string, string, bool, *CommonError) {
	tickets, validLastRandId, position, errFetch := ticketFetcher.FetchTimeline(lastRandId)
	if errFetch != nil {
		return nil, validLastRandId, position, false, &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errFetch,
		}
	}

	totalReceivedItems := int64(len(tickets))
	if totalReceivedItems < definition.ItemPerPage {
		seedRequired, errCheck := ticketFetcher.IsTimelineSeedingRequired(totalReceivedItems)
		if errCheck != nil {
			return nil, validLastRandId, position, false, &CommonError{
				Status: fiber.StatusInternalServerError,
				Code:   "T500",
				Error:  errCheck,
			}
		}

		if seedRequired {
			return tickets, validLastRandId, position, true, nil
		}
	}

	return tickets, validLastRandId, position, false, nil
}

func FetchTimelineByReporter(lastRandId []string, reporterUUID string, ticketFetcher *lib.TicketFetcher) ([]lib.Ticket, string, string, bool, *CommonError) {
	tickets, validLastRandId, position, errFetch := ticketFetcher.FetchTimelineByReporter(lastRandId, reporterUUID)
	if errFetch != nil {
		return nil, validLastRandId, position, false, &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errFetch,
		}
	}

	totalReceivedItems := int64(len(tickets))
	if totalReceivedItems < definition.ItemPerPage {
		seedRequired, errCheck := ticketFetcher.IsTimelineSeedingRequired(totalReceivedItems)
		if errCheck != nil {
			return nil, validLastRandId, position, false, &CommonError{
				Status: fiber.StatusInternalServerError,
				Code:   "T500",
				Error:  errCheck,
			}
		}

		if seedRequired {
			return nil, validLastRandId, position, true, nil
		}
	}

	return tickets, validLastRandId, position, false, nil
}

func FetchSorted(ticketFetcher *lib.TicketFetcher) ([]lib.Ticket, bool, *CommonError) {
	tickets, errFetch := ticketFetcher.FetchSorted()
	if errFetch != nil {
		return nil, false, &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errFetch,
		}
	}
	if len(tickets) == 0 {
		isSeedRequired, errCheck := ticketFetcher.IsSortedSeedingRequired()
		if errCheck != nil {
			return nil, false, &CommonError{
				Status: fiber.StatusInternalServerError,
				Code:   "T500",
				Error:  errCheck,
			}
		}
		if isSeedRequired {
			return nil, true, nil
		}
	}

	return tickets, false, nil
}

func FetchSortedByReporter(reporterUUID string, ticketFetcher *lib.TicketFetcher) ([]lib.Ticket, bool, *CommonError) {
	tickets, errFetch := ticketFetcher.FetchSortedByReporter(reporterUUID)
	if errFetch != nil {
		return nil, false, &CommonError{
			Status: fiber.StatusInternalServerError,
			Code:   "T500",
			Error:  errFetch,
		}
	}
	if len(tickets) == 0 {
		isSeedRequired, errCheck := ticketFetcher.IsSortedByReporterSeedingRequired(reporterUUID)
		if errCheck != nil {
			return nil, false, &CommonError{
				Status: fiber.StatusInternalServerError,
				Code:   "T500",
				Error:  errCheck,
			}
		}
		if isSeedRequired {
			return nil, true, nil
		}
	}

	return tickets, false, nil
}
