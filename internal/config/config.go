package config

import "os"

const (
	defaultHTTPAddr         = ":8080"
	defaultLawDBPath        = "data/db/laws.db"
	defaultLawDetailJSONDir = "data/law_json"
)

type Config struct {
	HTTPAddr         string
	LawDBPath        string
	LawDetailJSONDir string
}

func Load() Config {
	return Config{
		HTTPAddr:         getEnv("HTTP_ADDR", defaultHTTPAddr),
		LawDBPath:        getEnv("LAW_DB_PATH", defaultLawDBPath),
		LawDetailJSONDir: getEnv("LAW_DETAIL_JSON_DIR", defaultLawDetailJSONDir),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}
