package repository

import (
	"context"
	"database/sql"
	"redifu-example/definition"
	"redifu-example/internal/model"
)

type CategoryRepository struct {
	db *sql.DB
}

func (c *CategoryRepository) Init(db *sql.DB) {
	c.db = db
}

func (c *CategoryRepository) FindByRandId(ctx context.Context, randId string) (*model.Category, error) {
	query := "SELECT * FROM category WHERE randid = $1"
	stmt, err := c.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, randId)
	category := model.NewCategory()
	errScan := row.Scan(&category.UUID, &category.RandId, &category.CreatedAt, &category.UpdatedAt, &category.Category)
	if errScan != nil {
		if errScan == sql.ErrNoRows {
			return nil, definition.NotFound
		}
		return nil, errScan
	}

	return category, nil
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	categoryRepository := &CategoryRepository{}
	categoryRepository.Init(db)
	return categoryRepository
}
