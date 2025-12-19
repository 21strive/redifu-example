package model

import "github.com/21strive/redifu"

type Ticket struct {
	*redifu.Record
	Description  string `json:"description"`
	ReporterUUID string `json:"reporter_uuid"`
	Resolved     bool   `json:"action_taken"`
}

func (t *Ticket) SetDescription(description string) {
	t.Description = description
}

func (t *Ticket) SetReporterUUID(reporterUUID string) {
	t.ReporterUUID = reporterUUID
}

func (t *Ticket) SetResolved() {
	t.Resolved = true
}

func NewTicket() *Ticket {
	ticket := &Ticket{}
	redifu.InitRecord(ticket)
	return ticket
}
