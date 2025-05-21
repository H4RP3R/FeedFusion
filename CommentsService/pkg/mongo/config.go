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
	User   string
	Pass   string
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
	conf.User = os.Getenv("MONGO_USER")
	conf.Pass = os.Getenv("MONGO_PASS")

	return conf, nil
}

func (c *Config) conString() string {
	if c.User != "" && c.Pass != "" {
		return fmt.Sprintf("mongodb://%s:%s@%s:%s/", c.User, c.Pass, c.Host, c.Port)
	}
	return fmt.Sprintf("mongodb://%s:%s/", c.Host, c.Port)
	// u := os.Getenv("MONGODB_URL")
	// return fmt.Sprintf(u)
}

func (c *Config) Options() *options.ClientOptions {
	return options.Client().ApplyURI(c.conString())
}
