package postgres

import "testing"

func TestConfig_isValid(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{
			name: "valid config",
			cfg: Config{
				User:     "user",
				Password: "password",
				Host:     "localhost",
				Port:     "5432",
				DBName:   "test",
			},
			want: true,
		},
		{
			name: "empty config",
			cfg: Config{
				User:     "",
				Password: "",
				Host:     "",
				Port:     "",
				DBName:   "",
			},
			want: false,
		},
		{
			name: "config with empty DBName",
			cfg: Config{
				User:     "user",
				Password: "password",
				Host:     "localhost",
				Port:     "5432",
				DBName:   "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsValid(); got != tt.want {
				t.Errorf("Config.isValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
