-- Создание таблицы users
CREATE TABLE IF NOT EXISTS users (
                                     id SERIAL PRIMARY KEY,
                                     username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
    );

-- Создание таблицы balances
CREATE TABLE IF NOT EXISTS balances (
                                        user_id INT REFERENCES users(id) ON DELETE CASCADE,
    currency VARCHAR(3) NOT NULL,
    amount DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    PRIMARY KEY (user_id, currency),
    CHECK (amount >= 0)
    );

-- Создание таблицы exchange_rates
CREATE TABLE IF NOT EXISTS exchange_rates (
                                              from_currency VARCHAR(3) NOT NULL,
    to_currency VARCHAR(3) NOT NULL,
    rate DOUBLE PRECISION NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (from_currency, to_currency),
    CHECK (rate > 0)
    );

-- Индексы для оптимизации запросов
CREATE INDEX IF NOT EXISTS idx_balances_user_id ON balances(user_id);
CREATE INDEX IF NOT EXISTS idx_exchange_rates_from_currency ON exchange_rates(from_currency);