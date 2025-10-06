package lib

import (
	"database/sql"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"redifu-example/definition"
)

type Account struct {
	*redifu.SQLItem
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewAccount() *Account {
	account := &Account{}
	redifu.InitSQLItem(account)
	return account
}

type AccountRepository struct {
	db   *sql.DB
	base *redifu.Base[Account]
}

func (ar *AccountRepository) FindByUUID(uuid string) (*Account, error) {
	// If not in cache, query from database
	query := "SELECT uuid, randid, name, email FROM account WHERE uuid = $1"
	row := ar.db.QueryRow(query, uuid)

	account := NewAccount()
	err := row.Scan(&account.UUID, &account.RandId, &account.Name, &account.Email)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (ar *AccountRepository) Update(account *Account) error {
	query := "UPDATE account SET name = $1, email = $2 WHERE uuid = $3"
	stmt, err := ar.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, errUpdate := stmt.Exec(account.Name, account.Email, account.GetUUID())
	if errUpdate != nil {
		return errUpdate
	}

	errSet := ar.base.Set(*account)
	if errSet != nil {
		return errSet
	}

	return nil
}

func NewAccountRepository(db *sql.DB, redisClient redis.UniversalClient) *AccountRepository {
	base := redifu.NewBase[Account](redisClient, "account:%s", definition.BaseTTL)
	accountRepository := &AccountRepository{}
	accountRepository.db = db
	accountRepository.base = base
	return accountRepository
}
