package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL string
	Port        string
	CORSOrigins []string
}

func Load() (Config, error) {
	uri := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if uri == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	origins := splitCSV(os.Getenv("CORS_ORIGINS"))
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173"}
	}

	return Config{
		DatabaseURL: uri,
		Port:        port,
		CORSOrigins: origins,
	}, nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
