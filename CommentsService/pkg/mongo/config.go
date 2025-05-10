package mongo

import (
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrConfParamMissing = fmt.Errorf("configuration parameter missing")

type Config struct {
	Host   string
	Port   string
	DBName string
}

func NewConfig() (*Config, error) {
	conf := new(Config)
	conf.Host = os.Getenv("MONGO_HOST")
	if conf.Host == "" {
		return nil, fmt.Errorf("%w: MONGO_HOST", ErrConfParamMissing)
	}
	conf.Port = os.Getenv("MONGO_PORT")
	if conf.Port == "" {
		return nil, fmt.Errorf("%w: MONGO_PORT", ErrConfParamMissing)
	}
	conf.DBName = os.Getenv("MONGO_DB_NAME")
	if conf.DBName == "" {
		return nil, fmt.Errorf("%w: MONGO_DB_NAME", ErrConfParamMissing)
	}

	return conf, nil
}

func (c *Config) conString() string {
	return fmt.Sprintf("mongodb://%s:%s/", c.Host, c.Port)
}

func (c *Config) Options() *options.ClientOptions {
	return options.Client().ApplyURI(c.conString())
}
