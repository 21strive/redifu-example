package model

import "github.com/21strive/redifu"

type Account struct {
	*redifu.Record
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewAccount() *Account {
	account := &Account{}
	redifu.InitRecord(account)
	return account
}
