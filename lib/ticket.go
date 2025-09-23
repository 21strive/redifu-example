package lib

import (
	"database/sql"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"redifu-example/definition"
)

type Ticket struct {
	*redifu.SQLItem
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
	redifu.InitSQLItem(ticket)
	return ticket
}

type TicketRepository struct {
	db                       *sql.DB
	base                     *redifu.Base[Ticket]
	timeline                 *redifu.Timeline[Ticket]
	timelineSeeder           *redifu.TimelineSQLSeeder[Ticket]
	timelineByReporter       *redifu.Timeline[Ticket]
	timelineByReporterSeeder *redifu.TimelineSQLSeeder[Ticket]
	sorted                   *redifu.Sorted[Ticket]
	sortedSeeder             *redifu.SortedSQLSeeder[Ticket]
	sortedByReporter         *redifu.Sorted[Ticket]
	sortedByReporterSeeder   *redifu.SortedSQLSeeder[Ticket]
}

func (t *TicketRepository) Init(
	db *sql.DB,
	base *redifu.Base[Ticket],
	timeline *redifu.Timeline[Ticket],
	timelineSeeder *redifu.TimelineSQLSeeder[Ticket],
	timelineByReporter *redifu.Timeline[Ticket],
	timelineByReporterSeeder *redifu.TimelineSQLSeeder[Ticket],
	sorted *redifu.Sorted[Ticket],
	sortedSeeder *redifu.SortedSQLSeeder[Ticket],
	sortedByReporter *redifu.Sorted[Ticket],
	sortedByReporterSeeder *redifu.SortedSQLSeeder[Ticket],
) {
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

func (t *TicketRepository) Create(ticket *Ticket) error {
	query := "INSERT INTO ticket (uuid, randid, created_at, updated_at, reporter_uuid, description, resolved) VALUES ($1, $2, $3, $4, $5, $6, $7)"
	stmt, err := t.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, errCreate := stmt.Exec(ticket.GetUUID(), ticket.GetRandId(), ticket.GetCreatedAt(), ticket.GetUpdatedAt(), ticket.ReporterUUID, ticket.Description, ticket.Resolved)
	if errCreate != nil {
		return errCreate
	}

	errSet := t.base.Set(*ticket)
	if errSet != nil {
		return errSet
	}

	t.timeline.AddItem(*ticket, nil)
	t.timelineByReporter.AddItem(*ticket, []string{ticket.ReporterUUID})
	t.sorted.AddItem(*ticket, nil)
	t.sorted.AddItem(*ticket, []string{ticket.ReporterUUID})

	return nil
}

func (t *TicketRepository) Update(ticket *Ticket) error {
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

	errSet := t.base.Set(*ticket)
	if errSet != nil {
		return errSet
	}

	return nil
}

func (t *TicketRepository) Delete(ticket *Ticket) error {
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

	errDel := t.base.Del(*ticket)
	if errDel != nil {
		return errDel
	}

	t.timeline.RemoveItem(*ticket, nil)
	t.timelineByReporter.RemoveItem(*ticket, []string{ticket.ReporterUUID})
	t.sorted.RemoveItem(*ticket, nil)
	t.sorted.RemoveItem(*ticket, []string{ticket.ReporterUUID})

	return nil
}

func (t *TicketRepository) FindByUUID(uuid string) (*Ticket, error) {
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

func (t *TicketRepository) FindByRandId(randid string) (*Ticket, error) {
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

func rowScanner(row *sql.Row) (Ticket, error) {
	ticket := NewTicket()
	errScan := row.Scan(&ticket.UUID, &ticket.RandId, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.ReporterUUID, &ticket.Description, &ticket.Resolved)
	return *ticket, errScan
}

func rowsScanner(rows *sql.Rows) (Ticket, error) {
	ticket := NewTicket()
	errScan := rows.Scan(&ticket.UUID, &ticket.RandId, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.ReporterUUID, &ticket.Description, &ticket.Resolved)
	return *ticket, errScan
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

func NewTicketRepository(db *sql.DB, redisClient redis.UniversalClient) *TicketRepository {
	base := redifu.NewBase[Ticket](redisClient, "ticket:%s", definition.BaseTTL)
	timeline := redifu.NewTimeline[Ticket](redisClient, base, "ticket-timeline", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	timelineSeeeder := redifu.NewTimelineSQLSeeder[Ticket](db, base, timeline)
	timelineByReporter := redifu.NewTimeline[Ticket](redisClient, base, "ticket-timeline:%s", definition.ItemPerPage, redifu.Descending, definition.SortedSetTTL)
	timelineByReporterSeeeder := redifu.NewTimelineSQLSeeder[Ticket](db, base, timelineByReporter)
	sorted := redifu.NewSorted[Ticket](redisClient, base, "ticket-sorted", definition.SortedSetTTL)
	sortedSeeeder := redifu.NewSortedSQLSeeder[Ticket](db, base, sorted)
	sortedByReporter := redifu.NewSorted[Ticket](redisClient, base, "ticket-sorted:%s", definition.SortedSetTTL)
	sortedByReporterSeeeder := redifu.NewSortedSQLSeeder[Ticket](db, base, sortedByReporter)

	ticketRepository := &TicketRepository{}
	ticketRepository.Init(
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

type TicketFetcher struct {
	base               *redifu.Base[Ticket]
	timeline           *redifu.Timeline[Ticket]
	timelineByReporter *redifu.Timeline[Ticket]
	sorted             *redifu.Sorted[Ticket]
	sortedByReporter   *redifu.Sorted[Ticket]
}

func (t *TicketFetcher) Init(base *redifu.Base[Ticket], timeline *redifu.Timeline[Ticket], sorted *redifu.Sorted[Ticket]) {
	t.base = base
	t.timeline = timeline
	t.sorted = sorted
}

func (t *TicketFetcher) Fetch(randid string) (*Ticket, error) {
	ticket, err := t.base.Get(randid)
	if err != nil {
		return nil, err
	}

	return &ticket, nil
}

func (t *TicketFetcher) IsBlank(randid string) (bool, error) {
	return t.base.IsBlank(randid)
}

func (t *TicketFetcher) FetchTimeline(lastRandId []string) ([]Ticket, string, string, error) {
	return t.timeline.Fetch(nil, lastRandId, nil, nil)
}

func (t *TicketFetcher) IsTimelineSeedingRequired(totalReceivedItem int64) (bool, error) {
	return t.timeline.RequriesSeeding(nil, totalReceivedItem)
}

func (t *TicketFetcher) FetchTimelineByReporter(lastRandId []string, reporterUUID string) ([]Ticket, string, string, error) {
	return t.timeline.Fetch([]string{reporterUUID}, lastRandId, nil, nil)
}

func (t *TicketFetcher) IsTimelineByReporterSeedingRequired(totalReceivedItem int64, reporterUUID string) (bool, error) {
	return t.timeline.RequriesSeeding([]string{reporterUUID}, totalReceivedItem)
}

func (t *TicketFetcher) FetchSorted() ([]Ticket, error) {
	return t.sorted.Fetch(nil, redifu.Descending, nil, nil)
}

func (t *TicketFetcher) IsSortedSeedingRequired() (bool, error) {
	return t.sorted.RequiresSeeding(nil)
}

func (t *TicketFetcher) FetchSortedByReporter(reporterUUID string) ([]Ticket, error) {
	return t.sorted.Fetch([]string{reporterUUID}, redifu.Descending, nil, nil)
}

func (t *TicketFetcher) IsSortedByReporterSeedingRequired(reporterUUID string) (bool, error) {
	return t.sorted.RequiresSeeding([]string{reporterUUID})
}
