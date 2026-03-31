package store

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Postgres struct {
	DB *sqlx.DB
}

func NewPostgres(databaseURL string) (*Postgres, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return &Postgres{DB: db}, nil
}

func (p *Postgres) Close() error {
	return p.DB.Close()
}
