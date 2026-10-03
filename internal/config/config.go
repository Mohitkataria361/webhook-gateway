package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	// PostgreSQL
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisAddr     string
	RedisPassword string

	// RabbitMQ
	RabbitMQURL string

	// Application
	APIPort            string
	WorkerConcurrency  int
	MaxRetries         int
	BaseDelayMS        int

	// Circuit Breaker
	CBThreshold         int // failure % to trip breaker
	CBWindow            int // number of requests in sliding window
	CBOpenDurationMin   int // minutes circuit stays OPEN
}

// Load reads environment variables (from .env if present) and returns a Config.
func Load() *Config {
	// Load .env file if it exists (non-fatal if missing — rely on real env).
	if err := godotenv.Load(); err != nil {
		log.Println("[config] No .env file found, relying on OS environment variables")
	}

	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "webhook_user"),
		DBPassword: getEnv("DB_PASSWORD", "webhook_pass"),
		DBName:     getEnv("DB_NAME", "webhook_gateway"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),

		APIPort:           getEnv("API_PORT", "8080"),
		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 10),
		MaxRetries:        getEnvInt("MAX_RETRIES", 5),
		BaseDelayMS:       getEnvInt("BASE_DELAY_MS", 1000),

		CBThreshold:       getEnvInt("CIRCUIT_BREAKER_THRESHOLD", 50),
		CBWindow:          getEnvInt("CIRCUIT_BREAKER_WINDOW", 100),
		CBOpenDurationMin: getEnvInt("CIRCUIT_BREAKER_OPEN_DURATION_MIN", 15),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
