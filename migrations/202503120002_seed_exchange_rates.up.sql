-- Начальные данные для курсов валют
INSERT INTO exchange_rates (from_currency, to_currency, rate, updated_at)
VALUES
    ('USD', 'RUB', 90.0, NOW()),
    ('USD', 'EUR', 0.85, NOW()),
    ('RUB', 'USD', 0.011, NOW()),
    ('RUB', 'EUR', 0.009, NOW()),
    ('EUR', 'USD', 1.18, NOW()),
    ('EUR', 'RUB', 105.0, NOW())
    ON CONFLICT (from_currency, to_currency) DO NOTHING;