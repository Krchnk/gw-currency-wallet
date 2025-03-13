POST /api/v1/register ```- Регистрация пользователя```

POST /api/v1/login ```- Авторизация пользователя```

GET /api/v1/balance ```- Получение баланса (требуется JWT)```

POST /api/v1/wallet/deposit ```- Пополнение счета (требуется JWT)```

POST /api/v1/wallet/withdraw ```- Вывод средств (требуется JWT)```

GET /api/v1/exchange/rates ```- Получение курсов валют (требуется JWT)```

POST /api/v1/exchange ```- Обмен валют (требуется JWT)```


```миграции```

migrate -database "postgres://postgres:password@localhost:5432/wallet?sslmode=disable" -path migrations up