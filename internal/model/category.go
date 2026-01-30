package model

import "github.com/21strive/redifu"

type Category struct {
	*redifu.Record
	Category string
}

func (c *Category) SetCategory(category string) {
	c.Category = category
}

func NewCategory() *Category {
	category := &Category{}
	redifu.InitRecord(category)
	return category
}
