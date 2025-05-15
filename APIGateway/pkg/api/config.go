package api

import (
	"encoding/json"
	"os"
)

type Config struct {
	Services map[string]Service
}

type Service struct {
	URL  string
	Name string
}

func loadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
