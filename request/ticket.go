package request

type CreateTicketRequest struct {
	Description  string `json:"description"`
	ReporterUUID string `json:"reporter_uuid"`
}

type UpdateTicketDescriptionRequest struct {
	TicketUUID  string `json:"ticket_uuid"`
	Description string `json:"description"`
}
