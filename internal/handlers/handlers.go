package handlers

import (
	"fmt"
	"github.com/Krchnk/gw-currency-wallet/internal/config"
	"github.com/Krchnk/gw-currency-wallet/internal/storages"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"os"
	"time"
)

var logger = logrus.New()

func init() {
	logger.SetFormatter(&logrus.JSONFormatter{})
	if lvl, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL")); err == nil {
		logger.SetLevel(lvl)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
}

type Handler struct {
	store storages.Storage
	cfg   config.Config
	cache *cache.Cache
}

func NewHandler(store storages.Storage, cfg config.Config) *Handler {
	return &Handler{
		store: store,
		cfg:   cfg,
		cache: cache.New(5*time.Minute, 10*time.Minute),
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	if err := c.BindJSON(&req); err != nil {
		logger.WithError(err).Error("failed to bind registration request")
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	logger.WithFields(logrus.Fields{
		"username": req.Username,
		"email":    req.Email,
	}).Info("registration attempt")

	err := h.store.RegisterUser(req.Username, req.Password, req.Email)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"username": req.Username,
			"email":    req.Email,
		}).WithError(err).Error("user registration failed")
		c.JSON(400, gin.H{"error": "Username or email already exists"})
		return
	}

	logger.WithField("username", req.Username).Info("user registered successfully")
	c.JSON(201, gin.H{"message": "User registered successfully"})
}

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BindJSON(&req); err != nil {
		logger.WithError(err).Error("failed to bind login request")
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	logger.WithField("username", req.Username).Info("login attempt")

	user, err := h.store.GetUser(req.Username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		logger.WithField("username", req.Username).Error("invalid username or password")
		c.JSON(401, gin.H{"error": "Invalid username or password"})
		return
	}

	token, err := h.generateJWT(user.ID)
	if err != nil {
		logger.WithError(err).Error("failed to generate JWT")
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	logger.WithField("username", req.Username).Info("login successful")
	c.JSON(200, gin.H{"token": token})
}

func (h *Handler) GetBalance(c *gin.Context) {
	userID := c.GetString("user_id")

	logger.WithField("user_id", userID).Info("getting balance")

	balance, err := h.store.GetBalance(userID)
	if err != nil {
		logger.WithField("user_id", userID).WithError(err).Error("failed to get balance")
		c.JSON(500, gin.H{"error": "Failed to retrieve balance"})
		return
	}

	logger.WithFields(logrus.Fields{
		"user_id": userID,
		"balance": balance,
	}).Info("balance retrieved")
	c.JSON(200, gin.H{"balance": balance})
}

func (h *Handler) Deposit(c *gin.Context) {
	var req struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}

	if err := c.BindJSON(&req); err != nil {
		logger.WithError(err).Error("failed to bind deposit request")
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	userID := c.GetString("user_id")
	logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"amount":   req.Amount,
		"currency": req.Currency,
	}).Info("deposit attempt")

	if req.Amount <= 0 || !isValidCurrency(req.Currency) {
		logger.WithFields(logrus.Fields{
			"amount":   req.Amount,
			"currency": req.Currency,
		}).Error("invalid amount or currency")
		c.JSON(400, gin.H{"error": "Invalid amount or currency"})
		return
	}

	err := h.store.Deposit(userID, req.Currency, req.Amount)
	if err != nil {
		logger.WithError(err).Error("deposit failed")
		c.JSON(500, gin.H{"error": "Failed to deposit"})
		return
	}

	balance, _ := h.store.GetBalance(userID)
	logger.WithFields(logrus.Fields{
		"user_id": userID,
		"balance": balance,
	}).Info("deposit successful")
	c.JSON(200, gin.H{
		"message":     "Account topped up successfully",
		"new_balance": balance,
	})
}

func (h *Handler) Withdraw(c *gin.Context) {
	var req struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}

	if err := c.BindJSON(&req); err != nil {
		logger.WithError(err).Error("failed to bind withdraw request")
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	userID := c.GetString("user_id")
	logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"amount":   req.Amount,
		"currency": req.Currency,
	}).Info("withdraw attempt")

	if req.Amount <= 0 || !isValidCurrency(req.Currency) {
		logger.WithFields(logrus.Fields{
			"amount":   req.Amount,
			"currency": req.Currency,
		}).Error("invalid amount or currency")
		c.JSON(400, gin.H{"error": "Invalid amount or currency"})
		return
	}

	err := h.store.Withdraw(userID, req.Currency, req.Amount)
	if err != nil {
		logger.WithError(err).Error("withdraw failed")
		c.JSON(400, gin.H{"error": "Insufficient funds or invalid amount"})
		return
	}

	balance, _ := h.store.GetBalance(userID)
	logger.WithFields(logrus.Fields{
		"user_id": userID,
		"balance": balance,
	}).Info("withdraw successful")
	c.JSON(200, gin.H{
		"message":     "Withdrawal successful",
		"new_balance": balance,
	})
}

func (h *Handler) GetRates(c *gin.Context) {
	userID := c.GetString("user_id")
	logger.WithField("user_id", userID).Info("getting exchange rates")

	rates, err := h.store.GetExchangeRates()
	if err != nil {
		logger.WithError(err).Error("failed to get exchange rates from database")
		c.JSON(500, gin.H{"error": "Failed to retrieve exchange rates"})
		return
	}

	logger.WithFields(logrus.Fields{
		"user_id": userID,
		"rates":   rates,
	}).Info("exchange rates retrieved")
	c.JSON(200, gin.H{"rates": rates})
}

func (h *Handler) Exchange(c *gin.Context) {
	var req struct {
		FromCurrency string  `json:"from_currency"`
		ToCurrency   string  `json:"to_currency"`
		Amount       float64 `json:"amount"`
	}

	if err := c.BindJSON(&req); err != nil {
		logger.WithError(err).Error("failed to bind exchange request")
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	userID := c.GetString("user_id")
	logger.WithFields(logrus.Fields{
		"user_id": userID,
		"from":    req.FromCurrency,
		"to":      req.ToCurrency,
		"amount":  req.Amount,
	}).Info("exchange request initiated")

	if !isValidCurrency(req.FromCurrency) || !isValidCurrency(req.ToCurrency) || req.Amount <= 0 {
		logger.WithFields(logrus.Fields{
			"from":   req.FromCurrency,
			"to":     req.ToCurrency,
			"amount": req.Amount,
		}).Error("invalid currencies or amount")
		c.JSON(400, gin.H{"error": "Invalid currencies or amount"})
		return
	}

	rate, err := h.getExchangeRate(req.FromCurrency, req.ToCurrency)
	if err != nil {
		logger.WithError(err).Error("failed to get exchange rate")
		c.JSON(500, gin.H{"error": "Failed to get exchange rate"})
		return
	}

	if err := h.store.Exchange(userID, req.FromCurrency, req.ToCurrency, req.Amount, rate); err != nil {
		logger.WithError(err).Error("exchange operation failed")
		c.JSON(400, gin.H{"error": "Insufficient funds or invalid currencies"})
		return
	}

	balance, _ := h.store.GetBalance(userID)
	logger.WithFields(logrus.Fields{
		"user_id": userID,
		"balance": balance,
	}).Info("exchange completed successfully")

	c.JSON(200, gin.H{
		"message":          "Exchange successful",
		"exchanged_amount": req.Amount * rate,
		"new_balance":      balance,
	})
}

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" || len(tokenStr) < 7 || tokenStr[:7] != "Bearer " {
			logger.Error("missing or invalid Authorization header")
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		tokenStr = tokenStr[7:]
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(h.cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			logger.WithError(err).Error("invalid JWT token")
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userID := claims["user_id"].(string)
			c.Set("user_id", userID)
			logger.WithField("user_id", userID).Info("user authenticated")
			c.Next()
		} else {
			logger.Error("failed to parse JWT claims")
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
		}
	}
}

func (h *Handler) getExchangeRate(from, to string) (float64, error) {
	cacheKey := from + "_" + to
	if cached, found := h.cache.Get(cacheKey); found {
		logger.WithFields(logrus.Fields{
			"from": from,
			"to":   to,
		}).Info("rate retrieved from cache")
		return cached.(float64), nil
	}

	rate, err := h.store.GetExchangeRate(from, to)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"from": from,
			"to":   to,
		}).WithError(err).Error("failed to get rate from database")
		return 0, err
	}

	h.cache.Set(cacheKey, rate, 5*time.Minute)
	logger.WithFields(logrus.Fields{
		"from": from,
		"to":   to,
		"rate": rate,
	}).Info("rate retrieved from database and cached")
	return rate, nil
}

func (h *Handler) generateJWT(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": fmt.Sprintf("%d", userID),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	return token.SignedString([]byte(h.cfg.JWTSecret))
}

func isValidCurrency(currency string) bool {
	validCurrencies := map[string]bool{"USD": true, "RUB": true, "EUR": true}
	return validCurrencies[currency]
}
