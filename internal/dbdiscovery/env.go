package dbdiscovery

import (
	"bufio"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type envDiscoverer struct{}

func init() {
	RegisterDiscoverer(&envDiscoverer{})
}

func (e *envDiscoverer) Name() string { return "env" }

func (e *envDiscoverer) Scan(projectDir string) ([]DiscoveredDB, error) {
	var results []DiscoveredDB
	files := []string{".env", ".env.local"}
	for _, f := range files {
		path := filepath.Join(projectDir, f)
		vars, err := parseEnvFile(path)
		if err != nil {
			continue
		}
		results = append(results, discoverFromEnvVars(vars, path)...)
	}
	return results, nil
}

func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)
		if key != "" {
			vars[key] = val
		}
	}
	return vars, scanner.Err()
}

func discoverFromEnvVars(vars map[string]string, source string) []DiscoveredDB {
	var results []DiscoveredDB

	if v, ok := vars["DATABASE_URL"]; ok && v != "" {
		if db := parseDatabaseURL(v, source, "env/DATABASE_URL"); db != nil {
			results = append(results, *db)
		}
	}

	for _, key := range []string{"REDIS_URL", "REDISCLOUD_URL"} {
		if v, ok := vars[key]; ok && v != "" {
			dsn := v
			u, err := url.Parse(v)
			if err == nil && u.Host != "" {
				dsn = u.Host
				if u.User != nil {
					dsn = u.User.Username() + "@" + dsn
				}
			}
			results = append(results, DiscoveredDB{
				Name:   "env/" + key,
				Driver: "redis",
				DSN:    dsn,
				Source: source,
			})
		}
	}

	for _, key := range []string{"MONGODB_URI", "MONGO_URL"} {
		if v, ok := vars[key]; ok && v != "" {
			results = append(results, DiscoveredDB{
				Name:   "env/" + key,
				Driver: "mongo",
				DSN:    v,
				Source: source,
			})
		}
	}

	host := vars["DB_HOST"]
	port := vars["DB_PORT"]
	user := vars["DB_USER"]
	pass := vars["DB_PASSWORD"]
	name := vars["DB_NAME"]
	driver := vars["DB_DRIVER"]

	if host != "" && driver != "" {
		dsn := buildDSN(driver, host, port, user, pass, name)
		results = append(results, DiscoveredDB{
			Name:   "env/DB_HOST",
			Driver: driver,
			DSN:    dsn,
			Source: source,
		})
	}

	return results
}

func parseDatabaseURL(raw, source, name string) *DiscoveredDB {
	if strings.HasPrefix(raw, "sqlite:///") {
		return &DiscoveredDB{
			Name:   name,
			Driver: "sqlite",
			DSN:    strings.TrimPrefix(raw, "sqlite:///"),
			Source: source,
		}
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil
	}

	var driver string
	switch u.Scheme {
	case "postgresql", "postgres":
		driver = "postgres"
	case "mysql":
		driver = "mysql"
	default:
		return nil
	}

	return &DiscoveredDB{
		Name:   name,
		Driver: driver,
		DSN:    raw,
		Source: source,
	}
}

func buildDSN(driver, host, port, user, pass, dbname string) string {
	switch driver {
	case "postgres":
		dsn := "host=" + host
		if port != "" {
			dsn += " port=" + port
		}
		if user != "" {
			dsn += " user=" + user
		}
		if pass != "" {
			dsn += " password=" + pass
		}
		if dbname != "" {
			dsn += " dbname=" + dbname
		}
		dsn += " sslmode=disable"
		return dsn
	case "mysql":
		dsn := ""
		if user != "" {
			dsn += user
			if pass != "" {
				dsn += ":" + pass
			}
			dsn += "@"
		}
		dsn += "tcp(" + host
		if port != "" {
			dsn += ":" + port
		}
		dsn += ")"
		if dbname != "" {
			dsn += "/" + dbname
		}
		return dsn
	default:
		return host
	}
}
