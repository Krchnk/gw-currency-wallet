package storages

type Storage interface {
	RegisterUser(username, password, email string) error
	GetUser(username string) (User, error)
	GetBalance(userID string) (map[string]float64, error)
	Deposit(userID, currency string, amount float64) error
	Withdraw(userID, currency string, amount float64) error
	Exchange(userID, fromCurrency, toCurrency string, amount, rate float64) error
	GetExchangeRates() (map[string]float64, error)
	GetExchangeRate(from, to string) (float64, error)
}

type User struct {
	ID           int
	Username     string
	PasswordHash string
	Email        string
}
