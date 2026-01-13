package service

import (
	"context"
	"database/sql"
	"github.com/redis/go-redis/v9"
	"redifu-example/internal/fetcher"
	"redifu-example/internal/model"
	"redifu-example/internal/repository"
)

type AccountService struct {
	accountRepository *repository.AccountRepository
	accountFetcher    *fetcher.AccountFetcher
}

func (s *AccountService) InitRepository(db *sql.DB, redisClient redis.UniversalClient) {
	accountRepository := repository.NewAccountRepository(db, redisClient)
	s.accountRepository = accountRepository
}

func (s *AccountService) InitFetcher(redisClient redis.UniversalClient) {
	accountFetcher := fetcher.NewAccountFetcher(redisClient)
	s.accountFetcher = accountFetcher
}

func (s *AccountService) Create(ctx context.Context, name, email string) error {
	account := model.NewAccount()
	account.Name = name
	account.Email = email

	return s.accountRepository.Create(ctx, account)
}

func (s *AccountService) SeedAccountByUUID(ctx context.Context, accountUUID string) error {
	return s.accountRepository.SeedByUUID(ctx, accountUUID)
}

func (s *AccountService) GetAccountByUUID(ctx context.Context, accountUUID string) (*model.Account, error) {
	return s.accountFetcher.FetchByUUID(ctx, accountUUID)
}

func NewAccountService() *AccountService {
	return &AccountService{}
}
