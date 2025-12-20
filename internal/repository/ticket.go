package repository

import (
	"database/sql"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"log"
	"redifu-example/definition"
	"redifu-example/internal/model"
)

type TicketRepository struct {
	client                 redis.UniversalClient
	db                     *sql.DB
	base                   *redifu.Base[*model.Ticket]
	timeline               *redifu.Timeline[*model.Ticket]
	timelineSeeder         *redifu.TimelineSeeder[*model.Ticket]
	sortedByReporter       *redifu.Sorted[*model.Ticket]
	sortedByReporterSeeder *redifu.SortedSeeder[*model.Ticket]
}

func (t *TicketRepository) Init(
	redis redis.UniversalClient,
	db *sql.DB,
	base *redifu.Base[*model.Ticket],
	timeline *redifu.Timeline[*model.Ticket],
	timelineSeeder *redifu.TimelineSeeder[*model.Ticket],
	sortedByReporter *redifu.Sorted[*model.Ticket],
	sortedByReporterSeeder *redifu.SortedSeeder[*model.Ticket],
) {
	createTable := `
		CREATE TABLE IF NOT EXISTS ticket ( 
		    uuid varchar(36), 
		    randid varchar(16), 
		    created_at timestamp, 
		    updated_at timestamp, 
		    reporter_uuid varchar(36), 
		    description text, 
		    resolved bool 
	  	);
	`
	_, errCreateTable := db.Exec(createTable)
	if errCreateTable != nil {
		log.Fatal(errCreateTable)
	}

	t.client = redis
	t.db = db
	t.base = base
	t.timeline = timeline
	t.timelineSeeder = timelineSeeder
	t.sortedByReporter = sortedByReporter
	t.sortedByReporterSeeder = sortedByReporterSeeder
}

func (t *TicketRepository) Create(ticket *model.Ticket) error {
	query := "INSERT INTO ticket (uuid, randid, created_at, updated_at, description, resolved, account_uuid) VALUES ($1, $2, $3, $4, $5, $6, $7)"
	stmt, err := t.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, errCreate := stmt.Exec(ticket.GetUUID(), ticket.GetRandId(), ticket.GetCreatedAt(), ticket.GetUpdatedAt(), ticket.Description, ticket.Resolved, ticket.AccountUUID)
	if errCreate != nil {
		return errCreate
	}

	t.timeline.AddItem(ticket, nil)
	t.sortedByReporter.AddItem(ticket, []string{ticket.AccountUUID})

	return nil
}

func (t *TicketRepository) Update(ticket *model.Ticket) error {
	query := "UPDATE ticket SET description = $1, updated_at = $2, resolved = $3 WHERE uuid = $4"
	stmt, err := t.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, errUpdate := stmt.Exec(ticket.Description, ticket.GetUpdatedAt(), ticket.Resolved, ticket.GetUUID())
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

	return ticket, errScan
}

func rowScanner(row *sql.Row) (*model.Ticket, error) {
	ticket := model.NewTicket()
	errScan := row.Scan(&ticket.UUID, &ticket.RandId, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.Description, &ticket.Resolved, &ticket.AccountUUID)
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
		&ticket.Description,
		&ticket.Resolved,
		&ticket.AccountUUID,
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
		errSet := relation["account"].SetItem(*account)
		if errSet != nil {
			return ticket, errSet
		}
	}

	return ticket, nil
}

func (t *TicketRepository) SeedByRandId(randId string) error {
	ticket, errFind := t.FindByRandId(randId)
	if errFind != nil {
		return errFind
	}
	if ticket == nil {
		return t.base.SetBlank(randId)
	}

	errSet := t.base.Upsert(ticket)
	if errSet != nil {
		return errSet
	}

	return nil
}

func (t *TicketRepository) SeedTimeline(subtraction int64, lastRandId string) error {

	rowQuery := `
		  SELECT t.*, a.* 
		  FROM ticket t
		  LEFT JOIN account a ON t.account_uuid = a.uuid
		  WHERE t.randid = $1
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

func (t *TicketRepository) SeedSortedByReporter(reporterUUID string) error {
	query := "SELECT * FROM ticket WHERE account_uuid = $1"
	return t.sortedByReporterSeeder.SeedWithRelation(query, ticketScanner, []interface{}{reporterUUID}, []string{reporterUUID})
}

func NewTicketRepository(db *sql.DB, redisClient redis.UniversalClient) *TicketRepository {
	base := redifu.NewBase[*model.Ticket](redisClient, "ticket:%s", definition.BaseTTL)
	timeline := redifu.NewTimeline[*model.Ticket](redisClient, base, "ticket-timeline", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	timelineSeeder := redifu.NewTimelineSeeder[*model.Ticket](redisClient, db, base, timeline)
	sortedByAccount := redifu.NewSorted[*model.Ticket](redisClient, base, "ticket-sorted-by-account", definition.SortedSetTTL)
	sortedByReporterSeeder := redifu.NewSortedSeeder[*model.Ticket](redisClient, db, base, sortedByAccount)

	ticketRepository := &TicketRepository{}
	ticketRepository.Init(
		redisClient,
		db,
		base,
		timeline,
		timelineSeeder,
		sortedByAccount,
		sortedByReporterSeeder)

	return ticketRepository
}
