package config

import "os"

type Config struct {
	// MongoURI is the MongoDB connection string, e.g. mongodb://host:27017.
	MongoURI string
	// MongoDatabase is the database name used on the connected client.
	MongoDatabase string
	// Port is the HTTP listen port without colon.
	Port string
}

// Load reads configuration from the environment. Defaults live here — next
// to the code that enforces them — not in the database or deployment files.
func Load() Config {
	return Config{
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase: getEnv("MONGO_DATABASE", "dev_colab_hub"),
		Port:          getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
