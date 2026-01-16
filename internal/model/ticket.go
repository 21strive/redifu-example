package model

import "github.com/21strive/redifu"

type Ticket struct {
	*redifu.Record
	Description   string `json:"description"`
	Resolved      bool   `json:"action_taken"`
	SecurityRisk  int64  `json:"security_risk"`
	AccountUUID   string `json:"account_uuid"`
	AccountRandId string `json:",omitempty"`
	Account       *Account
}

func (t *Ticket) SetDescription(description string) {
	t.Description = description
}

func (t *Ticket) SetAccountUUID(accountUUID string) {
	t.AccountUUID = accountUUID
}

func (t *Ticket) SetResolved() {
	t.Resolved = true
}

func (t *Ticket) SetSecurityRisk(risk int64) {
	t.SecurityRisk = risk
}

func NewTicket() *Ticket {
	ticket := &Ticket{}
	redifu.InitRecord(ticket)
	return ticket
}
