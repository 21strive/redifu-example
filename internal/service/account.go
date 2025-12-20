package service

import (
	"database/sql"
	"github.com/redis/go-redis/v9"
	"redifu-example/internal/repository"
)

type AccountService struct {
	accountRepository *repository.AccountRepository
}

func (s *AccountService) InitRepository(db *sql.DB, redisClient redis.UniversalClient) {
	accountRepository := repository.NewAccountRepository(db, redisClient)
	s.accountRepository = accountRepository
}

func (s *AccountService) SeedAccountByUUID(accountUUID string) error {
	return s.accountRepository.SeedByUUID(accountUUID)
}
