package main

type CreateTicketRequest struct {
	Description  string `json:"description"`
	ReporterUUID string `json:"reporter_uuid"`
}

type UpdateTicketDescriptionRequest struct {
	TicketUUID  string `json:"ticket_uuid"`
	Description string `json:"description"`
}

type UpdateAccountRequest struct {
	AccountUUID string `json:"account_uuid"`
	Name        string `json:"name"`
	Email       string `json:"email"`
}
