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
	client                   redis.UniversalClient
	db                       *sql.DB
	base                     *redifu.Base[model.Ticket]
	timeline                 *redifu.Timeline[model.Ticket]
	timelineSeeder           *redifu.TimelineSeeder[model.Ticket]
	timelineByReporter       *redifu.Timeline[model.Ticket]
	timelineByReporterSeeder *redifu.TimelineSeeder[model.Ticket]
	sorted                   *redifu.Sorted[model.Ticket]
	sortedSeeder             *redifu.SortedSeeder[model.Ticket]
	sortedByReporter         *redifu.Sorted[model.Ticket]
	sortedByReporterSeeder   *redifu.SortedSeeder[model.Ticket]
}

func (t *TicketRepository) Init(
	redis redis.UniversalClient,
	db *sql.DB,
	base *redifu.Base[model.Ticket],
	timeline *redifu.Timeline[model.Ticket],
	timelineSeeder *redifu.TimelineSeeder[model.Ticket],
	timelineByReporter *redifu.Timeline[model.Ticket],
	timelineByReporterSeeder *redifu.TimelineSeeder[model.Ticket],
	sorted *redifu.Sorted[model.Ticket],
	sortedSeeder *redifu.SortedSeeder[model.Ticket],
	sortedByReporter *redifu.Sorted[model.Ticket],
	sortedByReporterSeeder *redifu.SortedSeeder[model.Ticket],
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
	t.timelineByReporter = timelineByReporter
	t.timelineByReporterSeeder = timelineByReporterSeeder
	t.sorted = sorted
	t.sortedSeeder = sortedSeeder
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

	_, errCreate := stmt.Exec(ticket.GetUUID(), ticket.GetRandId(), ticket.GetCreatedAt(), ticket.GetUpdatedAt(), ticket.Description, ticket.Resolved, ticket.ReporterUUID)
	if errCreate != nil {
		return errCreate
	}

	t.timeline.AddItem(*ticket, nil)
	t.timelineByReporter.AddItem(*ticket, []string{ticket.ReporterUUID})
	t.sorted.AddItem(*ticket, nil)
	t.sorted.AddItem(*ticket, []string{ticket.ReporterUUID})

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

	errUpset := t.base.Upsert(*ticket)
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

	t.timeline.RemoveItem(*ticket, nil)
	t.timelineByReporter.RemoveItem(*ticket, []string{ticket.ReporterUUID})
	t.sorted.RemoveItem(*ticket, nil)
	t.sorted.RemoveItem(*ticket, []string{ticket.ReporterUUID})

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

	return &ticket, nil
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

	return &ticket, errScan
}

func rowScanner(row *sql.Row) (model.Ticket, error) {
	ticket := model.NewTicket()
	errScan := row.Scan(&ticket.UUID, &ticket.RandId, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.Description, &ticket.Resolved, &ticket.ReporterUUID)
	return *ticket, errScan
}

func rowsScanner(rows *sql.Rows) (model.Ticket, error) {
	ticket := model.NewTicket()
	errScan := rows.Scan(&ticket.UUID, &ticket.RandId, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.Description, &ticket.Resolved, &ticket.ReporterUUID)
	return *ticket, errScan
}

func (t *TicketRepository) SeedByRandId(randId string) error {
	ticket, errFind := t.FindByRandId(randId)
	if errFind != nil {
		return errFind
	}
	if ticket == nil {
		return t.base.SetBlank(randId)
	}

	errSet := t.base.Upsert(*ticket)
	if errSet != nil {
		return errSet
	}

	return nil
}

func (t *TicketRepository) SeedTimeline(subtraction int64, lastRandId string) error {
	rowQuery := "SELECT * FROM ticket WHERE randid = $1"
	firstPageQuery := "SELECT * FROM ticket ORDER BY created_at DESC"
	nextPageQuery := "SELECT * FROM ticket WHERE created_at < $1 ORDER BY created_at DESC"

	return t.timelineSeeder.SeedPartial(
		rowQuery,
		firstPageQuery,
		nextPageQuery,
		rowScanner,
		rowsScanner,
		nil,
		subtraction,
		lastRandId,
		nil,
	)
}

func (t *TicketRepository) SeedTimelineByReporter(subtraction int64, lastRandId string, reporterUUID string) error {
	rowQuery := "SELECT * FROM ticket WHERE randid = $1"
	firstPageQuery := "SELECT * FROM ticket ORDER BY created_at DESC"
	nextPageQuery := "SELECT * FROM ticket WHERE created_at < $1 ORDER BY created_at DESC"

	return t.timelineSeeder.SeedPartial(
		rowQuery,
		firstPageQuery,
		nextPageQuery,
		rowScanner,
		rowsScanner,
		[]interface{}{reporterUUID},
		subtraction,
		lastRandId,
		[]string{reporterUUID},
	)
}

func (t *TicketRepository) SeedSorted() error {
	query := "SELECT * FROM ticket"
	return t.sortedSeeder.Seed(query, rowsScanner, nil, nil)
}

func (t *TicketRepository) SeedSortedByReporter(reporterUUID string) error {
	query := "SELECT * FROM ticket WHERE account_uuid = $1"
	return t.sortedSeeder.Seed(query, rowsScanner, []interface{}{reporterUUID}, []string{reporterUUID})
}

func NewTicketRepository(db *sql.DB, redisClient redis.UniversalClient) *TicketRepository {
	base := redifu.NewBase[model.Ticket](redisClient, "ticket:%s", definition.BaseTTL)
	timeline := redifu.NewTimeline[model.Ticket](redisClient, base, "ticket-timeline", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	timelineSeeeder := redifu.NewTimelineSeeder[model.Ticket](redisClient, db, base, timeline)
	timelineByReporter := redifu.NewTimeline[model.Ticket](redisClient, base, "ticket-timeline:%s", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	timelineByReporterSeeeder := redifu.NewTimelineSeeder[model.Ticket](redisClient, db, base, timelineByReporter)
	sorted := redifu.NewSorted[model.Ticket](redisClient, base, "ticket-sorted", definition.SortedSetTTL)
	sortedSeeeder := redifu.NewSortedSeeder[model.Ticket](redisClient, db, base, sorted)
	sortedByReporter := redifu.NewSorted[model.Ticket](redisClient, base, "ticket-sorted:%s", definition.SortedSetTTL)
	sortedByReporterSeeeder := redifu.NewSortedSeeder[model.Ticket](redisClient, db, base, sortedByReporter)

	ticketRepository := &TicketRepository{}
	ticketRepository.Init(
		redisClient,
		db,
		base,
		timeline,
		timelineSeeeder,
		timelineByReporter,
		timelineByReporterSeeeder,
		sorted,
		sortedSeeeder,
		sortedByReporter,
		sortedByReporterSeeeder)

	return ticketRepository
}
