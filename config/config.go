package config

import (
	"os"
	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	MongoURI   string
	JWTSecret  string
}

func Load() *Config {
	godotenv.Load()
	return &Config{
		Port:      getEnv("PORT", "8080"),
		MongoURI:  getEnv("MONGO_URI", "mongodb://localhost:27017"),
		JWTSecret: getEnv("JWT_SECRET", "secret"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}