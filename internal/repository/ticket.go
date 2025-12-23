package repository

import (
	"database/sql"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"redifu-example/definition"
	"redifu-example/internal/model"
)

type TicketRepository struct {
	db                           *sql.DB
	base                         *redifu.Base[*model.Ticket]
	timeline                     *redifu.Timeline[*model.Ticket]
	timelineSeeder               *redifu.TimelineSeeder[*model.Ticket]
	timelineBySecurityRisk       *redifu.Timeline[*model.Ticket]
	timelineBySecurityRiskSeeder *redifu.TimelineSeeder[*model.Ticket]
	sortedByReporter             *redifu.Sorted[*model.Ticket]
	sortedByReporterSeeder       *redifu.SortedSeeder[*model.Ticket]
	page                         *redifu.Page[*model.Ticket]
	pageSeeder                   *redifu.PageSeeder[*model.Ticket]
}

func (t *TicketRepository) Init(
	db *sql.DB,
	base *redifu.Base[*model.Ticket],
	timeline *redifu.Timeline[*model.Ticket],
	timelineSeeder *redifu.TimelineSeeder[*model.Ticket],
	timelineBySecurityRisk *redifu.Timeline[*model.Ticket],
	timelineBySecurityRiskSeeder *redifu.TimelineSeeder[*model.Ticket],
	sortedByReporter *redifu.Sorted[*model.Ticket],
	sortedByReporterSeeder *redifu.SortedSeeder[*model.Ticket],
	page *redifu.Page[*model.Ticket],
	pageSeeder *redifu.PageSeeder[*model.Ticket],
) {
	t.db = db
	t.base = base
	t.timeline = timeline
	t.timelineSeeder = timelineSeeder
	t.timelineBySecurityRisk = timelineBySecurityRisk
	t.timelineBySecurityRiskSeeder = timelineBySecurityRiskSeeder
	t.sortedByReporter = sortedByReporter
	t.sortedByReporterSeeder = sortedByReporterSeeder
	t.page = page
	t.pageSeeder = pageSeeder
}

func (t *TicketRepository) Create(ticket *model.Ticket) error {
	query := "INSERT INTO ticket (uuid, randid, created_at, updated_at, account_uuid, description, resolved, security_risk) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
	stmt, err := t.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, errCreate := stmt.Exec(ticket.GetUUID(), ticket.GetRandId(), ticket.GetCreatedAt(), ticket.GetUpdatedAt(), ticket.AccountUUID, ticket.Description, ticket.Resolved, ticket.SecurityRisk)
	if errCreate != nil {
		return errCreate
	}

	t.timeline.AddItem(ticket, nil)
	t.sortedByReporter.AddItem(ticket, []string{ticket.AccountUUID})
	t.base.Upsert(ticket)
	t.timelineBySecurityRisk.AddItem(ticket, nil)

	return nil
}

func (t *TicketRepository) Update(ticket *model.Ticket) error {
	query := "UPDATE ticket SET description = $1, resolved = $2, security_risk = $3, updated_at = $4 WHERE uuid = $5"
	stmt, err := t.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, errUpdate := stmt.Exec(ticket.Description, ticket.Resolved, ticket.SecurityRisk, ticket.GetUpdatedAt(), ticket.GetUUID())
	if errUpdate != nil {
		return errUpdate
	}

	errUpset := t.base.Upsert(ticket)
	return errUpset
}

func (t *TicketRepository) Delete(ticket *model.Ticket) error {
	query := "DELETE FROM ticket WHERE uuid = $1"
	stmt, err := t.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, errDelete := stmt.Exec(ticket.GetUUID())
	if errDelete != nil {
		return errDelete
	}

	t.timeline.RemoveItem(ticket, nil)
	t.sortedByReporter.RemoveItem(ticket, []string{ticket.AccountUUID})

	return nil
}

func (t *TicketRepository) FindByUUID(uuid string) (*model.Ticket, error) {
	query := "SELECT * FROM ticket WHERE uuid = $1"
	stmt, err := t.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(uuid)
	ticket, errScan := rowScanner(row)
	if errScan != nil {
		return nil, errScan
	}

	return ticket, nil
}

func (t *TicketRepository) FindByRandId(randid string) (*model.Ticket, error) {
	query := "SELECT * FROM ticket WHERE randid = $1"
	stmt, err := t.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(randid)
	ticket, errScan := rowScanner(row)
	if errScan != nil {
		if errScan == sql.ErrNoRows {
			return nil, definition.NotFound
		}
		return nil, errScan
	}

	return ticket, nil
}

func rowScanner(row *sql.Row) (*model.Ticket, error) {
	ticket := model.NewTicket()
	errScan := row.Scan(&ticket.UUID, &ticket.RandId, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.AccountUUID, &ticket.Description, &ticket.Resolved, &ticket.SecurityRisk)
	return ticket, errScan
}

func ticketScanner(rows *sql.Rows, relation map[string]redifu.Relation) (*model.Ticket, error) {
	account := model.NewAccount()
	ticket := model.NewTicket()

	// Use sql.Null* types for nullable fields from the join
	var accountUUID sql.NullString
	var accountRandId sql.NullString
	var accountCreatedAt sql.NullTime
	var accountUpdatedAt sql.NullTime
	var accountName sql.NullString
	var accountEmail sql.NullString

	errScan := rows.Scan(
		&ticket.UUID,
		&ticket.RandId,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
		&ticket.AccountUUID,
		&ticket.Description,
		&ticket.Resolved,
		&ticket.SecurityRisk,
		&accountUUID,
		&accountRandId,
		&accountCreatedAt,
		&accountUpdatedAt,
		&accountName,
		&accountEmail,
	)

	if errScan != nil {
		return ticket, errScan
	}

	// Only populate account if the join returned data (not NULL)
	if accountRandId.Valid {
		account.UUID = accountUUID.String
		account.RandId = accountRandId.String
		account.CreatedAt = accountCreatedAt.Time
		account.UpdatedAt = accountUpdatedAt.Time
		account.Name = accountName.String
		account.Email = accountEmail.String

		ticket.AccountRandId = account.RandId
		errSet := relation["account"].SetItem(account)
		if errSet != nil {
			return ticket, errSet
		}
	}

	return ticket, nil
}

func (t *TicketRepository) SeedTicket(randId string) error {
	ticket, errFind := t.FindByRandId(randId)
	if errFind != nil {
		if ticket == nil {
			t.base.SetBlank(randId)
		}
		return errFind
	}

	errSet := t.base.Upsert(ticket)
	if errSet != nil {
		return errSet
	}

	return nil
}

func (t *TicketRepository) SeedTickets(subtraction int64, lastRandId string) error {
	rowQuery := `
		  SELECT * FROM ticket
		  WHERE randid = $1
		`

	firstPageQuery := `
		  SELECT t.*, a.* 
		  FROM ticket t
		  LEFT JOIN account a ON t.account_uuid = a.uuid
		  ORDER BY t.created_at DESC
		`

	nextPageQuery := `
		  SELECT t.*, a.* 
		  FROM ticket t
		  LEFT JOIN account a ON t.account_uuid = a.uuid
		  WHERE t.created_at < $1 
		  ORDER BY t.created_at DESC
		`

	return t.timelineSeeder.SeedPartialWithRelation(
		rowQuery,
		firstPageQuery,
		nextPageQuery,
		rowScanner,
		ticketScanner,
		nil,
		subtraction,
		lastRandId,
		nil,
	)
}

func (t *TicketRepository) SeedTicketsBySecurityRisk(subtraction int64, lastRandId string) error {
	rowQuery := `
		  SELECT * FROM ticket
		  WHERE randid = $1
		`

	firstPageQuery := `
		  SELECT t.*, a.* 
		  FROM ticket t
		  LEFT JOIN account a ON t.account_uuid = a.uuid
		  ORDER BY t.security_risk DESC
		`

	nextPageQuery := `
		  SELECT t.*, a.* 
		  FROM ticket t
		  LEFT JOIN account a ON t.account_uuid = a.uuid
		  WHERE t.security_risk < $1 
		  ORDER BY t.security_risk DESC
		`

	return t.timelineBySecurityRiskSeeder.SeedPartialWithRelation(
		rowQuery,
		firstPageQuery,
		nextPageQuery,
		rowScanner,
		ticketScanner,
		nil,
		subtraction,
		lastRandId,
		nil,
	)
}

func (t *TicketRepository) SeedByAccount(reporterUUID string) error {
	query := "SELECT * FROM ticket WHERE account_uuid = $1"
	return t.sortedByReporterSeeder.SeedWithRelation(query, ticketScanner, []interface{}{reporterUUID}, []string{reporterUUID})
}

func (t *TicketRepository) SeedPage(page int64) error {
	query := `
		  SELECT t.*, a.* 
		  FROM ticket t
		  LEFT JOIN account a ON t.account_uuid = a.uuid
		  ORDER BY t.created_at DESC
		`

	return t.pageSeeder.SeedWithRelation(query, page, ticketScanner, nil, nil)
}

func NewTicketRepository(db *sql.DB, redisClient redis.UniversalClient) *TicketRepository {
	base := redifu.NewBase[*model.Ticket](redisClient, "ticket:%s", definition.BaseTTL)
	baseAccount := redifu.NewBase[*model.Account](redisClient, "account:%s", definition.BaseTTL)
	accountRelation := redifu.NewRelation[*model.Account](baseAccount, "Account", "AccountRandId")

	// Timeline - CreatedAt
	timeline := redifu.NewTimeline[*model.Ticket](redisClient, base, "ticket-timeline", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	timeline.AddRelation("account", accountRelation)
	timelineSeeder := redifu.NewTimelineSeeder[*model.Ticket](redisClient, db, base, timeline)

	// Timeline - Custom Scoring
	timelineBySecurityRisk := redifu.NewTimeline[*model.Ticket](redisClient, base, "ticket-timeline", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	timelineBySecurityRisk.AddRelation("account", accountRelation)
	timelineBySecurityRisk.SetSortingReference("SecurityRisk")
	timelineBySecurityRiskSeeder := redifu.NewTimelineSeeder[*model.Ticket](redisClient, db, base, timelineBySecurityRisk)
	timelineBySecurityRiskSeeder.SetSortingReference("SecurityRisk")

	// Sorted - CreatedAt
	sortedByAccount := redifu.NewSorted[*model.Ticket](redisClient, base, "ticket-sorted-by-account", definition.SortedSetTTL)
	sortedByAccount.AddRelation("account", accountRelation)
	sortedByReporterSeeder := redifu.NewSortedSeeder[*model.Ticket](redisClient, db, base, sortedByAccount)

	// Page - CreatedAt
	page := redifu.NewPage[*model.Ticket](redisClient, base, "ticket-page", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	page.AddRelation("account", accountRelation)
	pageSeeder := redifu.NewPageSeeder[*model.Ticket](redisClient, db, base, page)

	ticketRepository := &TicketRepository{}
	ticketRepository.Init(
		db,
		base,
		timeline,
		timelineSeeder,
		timelineBySecurityRisk,
		timelineBySecurityRiskSeeder,
		sortedByAccount,
		sortedByReporterSeeder,
		page,
		pageSeeder)

	return ticketRepository
}
