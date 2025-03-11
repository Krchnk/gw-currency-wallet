package postgres

import (
	"database/sql"
	"github.com/Krchnk/gw-currency-wallet/internal/config"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func NewStorage(cfg config.DBConfig) (*Storage, error) {
	connStr := cfg.ConnectionString()
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logrus.WithError(err).Error("failed to open database connection")
		return nil, err
	}

	if err := db.Ping(); err != nil {
		logrus.WithError(err).Error("failed to ping database")
		return nil, err
	}

	logrus.Info("database connection established")
	return &Storage{db: db}, nil
}
