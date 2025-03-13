package postgres

import (
	"database/sql"
	"errors"
	"github.com/Krchnk/gw-currency-wallet/internal/storages"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type Storage struct {
	db *sql.DB
}

func (s *Storage) RegisterUser(username, password, email string) error {
	var exists int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = $1 OR email = $2", username, email).Scan(&exists)
	if err != nil {
		logrus.WithError(err).Error("failed to check user existence")
		return err
	}
	if exists > 0 {
		logrus.WithFields(logrus.Fields{
			"username": username,
			"email":    email,
		}).Error("username or email already exists")
		return errors.New("username or email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logrus.WithError(err).Error("failed to hash password")
		return err
	}

	_, err = s.db.Exec(`
        INSERT INTO users (username, password_hash, email, created_at)
        VALUES ($1, $2, $3, NOW())`,
		username, string(hashedPassword), email)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"username": username,
			"email":    email,
		}).WithError(err).Error("failed to register user")
		return err
	}

	logrus.WithField("username", username).Info("user registered in database")
	return nil
}

func (s *Storage) GetUser(username string) (storages.User, error) {
	var user storages.User
	err := s.db.QueryRow(`
        SELECT id, username, password_hash, email
        FROM users
        WHERE username = $1`,
		username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			logrus.WithField("username", username).Error("user not found")
			return storages.User{}, errors.New("user not found")
		}
		logrus.WithField("username", username).WithError(err).Error("failed to get user")
		return storages.User{}, err
	}

	logrus.WithField("username", username).Info("user retrieved from database")
	return user, nil
}

func (s *Storage) GetBalance(userID string) (map[string]float64, error) {
	rows, err := s.db.Query(`
        SELECT currency, amount
        FROM balances
        WHERE user_id = $1`,
		userID)
	if err != nil {
		logrus.WithField("user_id", userID).WithError(err).Error("failed to query balance")
		return nil, err
	}
	defer rows.Close()

	balance := make(map[string]float64)
	for rows.Next() {
		var currency string
		var amount float64
		if err := rows.Scan(&currency, &amount); err != nil {
			logrus.WithField("user_id", userID).WithError(err).Error("failed to scan balance")
			return nil, err
		}
		balance[currency] = amount
	}

	for _, currency := range []string{"USD", "RUB", "EUR"} {
		if _, exists := balance[currency]; !exists {
			balance[currency] = 0.0
		}
	}

	logrus.WithFields(logrus.Fields{
		"user_id": userID,
		"balance": balance,
	}).Info("balance retrieved from database")
	return balance, nil
}

func (s *Storage) Deposit(userID, currency string, amount float64) error {
	tx, err := s.db.Begin()
	if err != nil {
		logrus.WithError(err).Error("failed to begin transaction for deposit")
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
        INSERT INTO balances (user_id, currency, amount)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, currency)
        DO UPDATE SET amount = balances.amount + EXCLUDED.amount`,
		userID, currency, amount)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id":  userID,
			"currency": currency,
			"amount":   amount,
		}).WithError(err).Error("failed to deposit amount")
		return err
	}

	if err := tx.Commit(); err != nil {
		logrus.WithError(err).Error("failed to commit deposit transaction")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"user_id":  userID,
		"currency": currency,
		"amount":   amount,
	}).Info("deposit completed in database")
	return nil
}

func (s *Storage) Withdraw(userID, currency string, amount float64) error {
	tx, err := s.db.Begin()
	if err != nil {
		logrus.WithError(err).Error("failed to begin transaction for withdraw")
		return err
	}
	defer tx.Rollback()

	var currentBalance float64
	err = tx.QueryRow(`
        SELECT amount
        FROM balances
        WHERE user_id = $1 AND currency = $2
        FOR UPDATE`,
		userID, currency).Scan(&currentBalance)
	if err != nil {
		if err == sql.ErrNoRows {
			currentBalance = 0.0
		} else {
			logrus.WithFields(logrus.Fields{
				"user_id":  userID,
				"currency": currency,
			}).WithError(err).Error("failed to get current balance for withdraw")
			return err
		}
	}

	if currentBalance < amount {
		logrus.WithFields(logrus.Fields{
			"user_id":         userID,
			"currency":        currency,
			"current_balance": currentBalance,
			"amount":          amount,
		}).Error("insufficient funds for withdraw")
		return errors.New("insufficient funds")
	}

	_, err = tx.Exec(`
        INSERT INTO balances (user_id, currency, amount)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, currency)
        DO UPDATE SET amount = balances.amount - EXCLUDED.amount`,
		userID, currency, amount)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id":  userID,
			"currency": currency,
			"amount":   amount,
		}).WithError(err).Error("failed to withdraw amount")
		return err
	}

	if err := tx.Commit(); err != nil {
		logrus.WithError(err).Error("failed to commit withdraw transaction")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"user_id":  userID,
		"currency": currency,
		"amount":   amount,
	}).Info("withdraw completed in database")
	return nil
}

func (s *Storage) Exchange(userID, fromCurrency, toCurrency string, amount, rate float64) error {
	tx, err := s.db.Begin()
	if err != nil {
		logrus.WithError(err).Error("failed to begin transaction for exchange")
		return err
	}
	defer tx.Rollback()

	var fromBalance float64
	err = tx.QueryRow(`
        SELECT amount
        FROM balances
        WHERE user_id = $1 AND currency = $2
        FOR UPDATE`,
		userID, fromCurrency).Scan(&fromBalance)
	if err != nil {
		if err == sql.ErrNoRows {
			fromBalance = 0.0
		} else {
			logrus.WithFields(logrus.Fields{
				"user_id":       userID,
				"from_currency": fromCurrency,
			}).WithError(err).Error("failed to get from balance for exchange")
			return err
		}
	}

	if fromBalance < amount {
		logrus.WithFields(logrus.Fields{
			"user_id":       userID,
			"from_currency": fromCurrency,
			"from_balance":  fromBalance,
			"amount":        amount,
		}).Error("insufficient funds for exchange")
		return errors.New("insufficient funds")
	}

	_, err = tx.Exec(`
        INSERT INTO balances (user_id, currency, amount)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, currency)
        DO UPDATE SET amount = balances.amount - EXCLUDED.amount`,
		userID, fromCurrency, amount)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id":       userID,
			"from_currency": fromCurrency,
			"amount":        amount,
		}).WithError(err).Error("failed to deduct from currency")
		return err
	}

	toAmount := amount * rate
	_, err = tx.Exec(`
        INSERT INTO balances (user_id, currency, amount)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, currency)
        DO UPDATE SET amount = balances.amount + EXCLUDED.amount`,
		userID, toCurrency, toAmount)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id":     userID,
			"to_currency": toCurrency,
			"amount":      toAmount,
		}).WithError(err).Error("failed to add to currency")
		return err
	}

	if err := tx.Commit(); err != nil {
		logrus.WithError(err).Error("failed to commit exchange transaction")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"user_id":       userID,
		"from_currency": fromCurrency,
		"to_currency":   toCurrency,
		"amount":        amount,
		"rate":          rate,
		"to_amount":     toAmount,
	}).Info("exchange completed in database")
	return nil
}

func (s *Storage) GetExchangeRates() (map[string]float64, error) {
	rows, err := s.db.Query(`
        SELECT to_currency, rate
        FROM exchange_rates
        WHERE from_currency = $1`,
		"USD")
	if err != nil {
		logrus.WithError(err).Error("failed to query exchange rates")
		return nil, err
	}
	defer rows.Close()

	rates := make(map[string]float64)
	rates["USD"] = 1.0 // USD как базовая валюта
	for rows.Next() {
		var currency string
		var rate float64
		if err := rows.Scan(&currency, &rate); err != nil {
			logrus.WithError(err).Error("failed to scan exchange rates")
			return nil, err
		}
		rates[currency] = rate
	}

	logrus.WithField("rates", rates).Info("exchange rates retrieved from database")
	return rates, nil
}

func (s *Storage) GetExchangeRate(from, to string) (float64, error) {
	var rate float64
	err := s.db.QueryRow(`
        SELECT rate
        FROM exchange_rates
        WHERE from_currency = $1 AND to_currency = $2`,
		from, to).Scan(&rate)
	if err != nil {
		if err == sql.ErrNoRows {
			logrus.WithFields(logrus.Fields{
				"from": from,
				"to":   to,
			}).Error("exchange rate not found")
			return 0, errors.New("exchange rate not found")
		}
		logrus.WithFields(logrus.Fields{
			"from": from,
			"to":   to,
		}).WithError(err).Error("failed to get exchange rate")
		return 0, err
	}

	logrus.WithFields(logrus.Fields{
		"from": from,
		"to":   to,
		"rate": rate,
	}).Info("exchange rate retrieved from database")
	return rate, nil
}
