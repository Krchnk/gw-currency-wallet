package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"os"
)

type Config struct {
	//HTTPPort  string
	DBConfig  DBConfig
	JWTSecret string
}

type DBConfig struct {
	Host               string
	Port               string
	User               string
	Password           string
	DBName             string
	SSLMode            string // Новое поле для sslmode
	SSLRootCert        string // Новое поле для sslrootcert
	TargetSessionAttrs string // Новое поле для target_session_attrs
}

func (d DBConfig) ConnectionString() string {
	// Формируем базовую строку подключения
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s target_session_attrs=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode, d.TargetSessionAttrs)

	// Если указан сертификат, добавляем его
	if d.SSLRootCert != "" {
		connStr += fmt.Sprintf(" sslrootcert=%s", d.SSLRootCert)
	}

	return connStr
}

func LoadConfig(path string) (Config, error) {
	if err := godotenv.Load(path); err != nil {
		logrus.WithError(err).Warn("failed to load config file, using env vars")
	}

	cfg := Config{
		//HTTPPort:  getEnv("HTTP_PORT", ":8080"),
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key"),
		DBConfig: DBConfig{
			Host:               getEnv("DB_HOST", "localhost"),
			Port:               getEnv("DB_PORT", "5432"),
			User:               getEnv("DB_USER", "postgres"),
			Password:           getEnv("DB_PASSWORD", "password"),
			DBName:             getEnv("DB_NAME", "wallet"),
			SSLMode:            getEnv("DB_SSLMODE", "verify-full"),             // Читаем DB_SSLMODE
			SSLRootCert:        getEnv("DB_SSLROOTCERT", ""),                    // Читаем DB_SSLROOTCERT
			TargetSessionAttrs: getEnv("DB_TARGET_SESSION_ATTRS", "read-write"), // Читаем DB_TARGET_SESSION_ATTRS
		},
	}
	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
