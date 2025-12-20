package repository

import (
	"context"
	"database/sql"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"redifu-example/definition"
	"redifu-example/internal/model"
)

type AccountRepository struct {
	redisClient redis.UniversalClient
	db          *sql.DB
	base        *redifu.Base[*model.Account]
}

func (ar *AccountRepository) Init(db *sql.DB, base *redifu.Base[*model.Account]) {
	ar.base = base
	ar.db = db
}

func (ar *AccountRepository) FindByUUID(accountUUID string) (*model.Account, error) {
	query := "SELECT uuid, randid, name, email FROM account WHERE uuid = $1"
	row := ar.db.QueryRow(query, accountUUID)

	account := model.NewAccount()
	err := row.Scan(&account.UUID, &account.RandId, &account.Name, &account.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, definition.NotFound
		}
		return nil, err
	}

	return account, nil
}

func (ar *AccountRepository) SeedByUUID(accountUUID string) error {
	accountFromDB, errFind := ar.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	errSet := ar.redisClient.Set(context.Background(), "account:pointer:"+accountUUID, accountFromDB.GetRandId(), definition.BaseTTL).Err()
	if errSet != nil {
		return errSet
	}

	return ar.base.Upsert(accountFromDB)
}

func (ar *AccountRepository) Update(account *model.Account) error {
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

	errSet := ar.base.Upsert(account)
	if errSet != nil {
		return errSet
	}

	return nil
}

func NewAccountRepository(db *sql.DB, redisClient redis.UniversalClient) *AccountRepository {
	base := redifu.NewBase[*model.Account](redisClient, "account:%s", definition.BaseTTL)
	accountRepository := &AccountRepository{}
	accountRepository.Init(db, base)
	return accountRepository
}
