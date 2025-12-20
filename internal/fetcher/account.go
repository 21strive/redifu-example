package fetcher

import (
	"context"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"redifu-example/definition"
	"redifu-example/internal/model"
)

type AccountFetcher struct {
	redisClient redis.UniversalClient
	base        *redifu.Base[*model.Account]
}

func (a *AccountFetcher) Init(base *redifu.Base[*model.Account]) {
	a.base = base
}

func (a *AccountFetcher) Fetch(accountRandId string) (*model.Account, error) {
	account, err := a.base.Get(accountRandId)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (a *AccountFetcher) IsBlank(accountRandId string) (bool, error) {
	return a.base.IsBlank(accountRandId)
}

func (a *AccountFetcher) FetchByUUID(accountUUID string) (*model.Account, error) {
	// resolve pointer
	errGet := a.redisClient.Get(context.Background(), "account:pointer:"+accountUUID)
	if errGet.Err() != nil {
		if errGet.Err() == redis.Nil {
			return nil, definition.NotFound
		}
		return nil, errGet.Err()
	}

	accountRandId := errGet.Val()
	isBlank, errCheck := a.IsBlank(accountRandId)
	if errCheck != nil {
		return nil, errCheck
	}
	if isBlank {
		return nil, definition.NotFound
	}

	return a.Fetch(accountRandId)
}

func NewAccountFetcher(redisClient redis.UniversalClient) *AccountFetcher {
	base := redifu.NewBase[*model.Account](redisClient, "account:%s", definition.BaseTTL)
	accountFetcher := &AccountFetcher{}
	accountFetcher.Init(base)
	return accountFetcher
}
