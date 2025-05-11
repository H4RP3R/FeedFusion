package postgres

import (
	"fmt"
	"strings"
)

type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	DBName   string
}

func (c *Config) ConString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", c.User, c.Password, c.Host, c.Port, c.DBName)
}

func (c Config) String() string {
	var sb strings.Builder
	for i := 0; i < len([]rune(c.Password)); i++ {
		sb.WriteString("*")
	}
	c.Password = sb.String()

	return fmt.Sprintf("%#v", c)
}
