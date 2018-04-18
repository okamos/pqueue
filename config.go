package pqueue

type config struct {
	m map[string]string
}

var cached = config{
	m: map[string]string{
		"psql_dsn": "host=localhost user=postgres dbname=postgres sslmode=disable",
	},
}

// GetConfig returns the configuration value of a key.
func GetConfig(key string) string {
	return cached.m[key]
}

// SetConfig sets the configuration value of a key.
func SetConfig(key string, v string) {
	cached.m[key] = v
}
