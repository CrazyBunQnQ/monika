package dbdiscovery

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type dockerDiscoverer struct{}

func init() {
	RegisterDiscoverer(&dockerDiscoverer{})
}

func (d *dockerDiscoverer) Name() string { return "docker" }

type composeFile struct {
	Services map[string]struct {
		Image       string            `yaml:"image"`
		Environment interface{}       `yaml:"environment"`
		Ports       []string          `yaml:"ports"`
	} `yaml:"services"`
}

func (d *dockerDiscoverer) Scan(projectDir string) ([]DiscoveredDB, error) {
	var results []DiscoveredDB
	files := []string{"docker-compose.yml", "docker-compose.yaml"}
	for _, f := range files {
		path := filepath.Join(projectDir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cf composeFile
		if err := yaml.Unmarshal(data, &cf); err != nil {
			continue
		}
		results = append(results, scanComposeServices(cf.Services, path)...)
	}
	return results, nil
}

func scanComposeServices(services map[string]struct {
	Image       string      `yaml:"image"`
	Environment interface{} `yaml:"environment"`
	Ports       []string    `yaml:"ports"`
}, source string) []DiscoveredDB {
	var results []DiscoveredDB
	for name, svc := range services {
		img := strings.ToLower(svc.Image)
		driver := driverFromImage(img)
		if driver == "" {
			continue
		}
		env := envToMap(svc.Environment)
		port := extractPort(svc.Ports, defaultPort(driver))
		dsn := buildDockerDSN(driver, env, port)
		results = append(results, DiscoveredDB{
			Name:   "docker/" + name,
			Driver: driver,
			DSN:    dsn,
			Source: source,
		})
	}
	return results
}

func driverFromImage(image string) string {
	if strings.Contains(image, "postgres") || strings.Contains(image, "postgresql") {
		return "postgres"
	}
	if strings.Contains(image, "mysql") {
		return "mysql"
	}
	if strings.Contains(image, "mongo") {
		return "mongo"
	}
	if strings.Contains(image, "redis") {
		return "redis"
	}
	return ""
}

func defaultPort(driver string) string {
	switch driver {
	case "postgres":
		return "5432"
	case "mysql":
		return "3306"
	case "redis":
		return "6379"
	case "mongo":
		return "27017"
	}
	return ""
}

func envToMap(env interface{}) map[string]string {
	m := make(map[string]string)
	switch v := env.(type) {
	case map[string]interface{}:
		for key, val := range v {
			m[key] = fmt.Sprintf("%v", val)
		}
	case []interface{}:
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				continue
			}
			idx := strings.Index(s, "=")
			if idx < 0 {
				continue
			}
			m[s[:idx]] = s[idx+1:]
		}
	}
	return m
}

func extractPort(ports []string, defaultP string) string {
	for _, p := range ports {
		hostPort := extractSinglePort(p)
		if hostPort == "" {
			continue
		}
		return hostPort
	}
	return defaultP
}

func extractSinglePort(mapping string) string {
	p := mapping
	if idx := strings.Index(p, "/"); idx >= 0 {
		p = p[:idx]
	}
	if strings.Contains(p, "-") {
		return ""
	}
	parts := strings.Split(p, ":")
	switch len(parts) {
	case 2:
		return parts[0]
	case 3:
		return parts[1]
	default:
		return ""
	}
}

func buildDockerDSN(driver string, env map[string]string, port string) string {
	host := "localhost"
	switch driver {
	case "postgres":
		user := env["POSTGRES_USER"]
		pass := env["POSTGRES_PASSWORD"]
		db := env["POSTGRES_DB"]
		if db == "" {
			db = "postgres"
		}
		dsn := fmt.Sprintf("host=%s port=%s", host, port)
		if user != "" {
			dsn += fmt.Sprintf(" user=%s", user)
		}
		if pass != "" {
			dsn += fmt.Sprintf(" password=%s", quoteDSNValue(pass))
		}
		dsn += fmt.Sprintf(" dbname=%s sslmode=prefer", db)
		return dsn
	case "mysql":
		user := env["MYSQL_USER"]
		pass := env["MYSQL_PASSWORD"]
		db := env["MYSQL_DATABASE"]
		if user == "" {
			rootPass := env["MYSQL_ROOT_PASSWORD"]
			if rootPass != "" {
				user = "root"
				pass = rootPass
			}
		}
		if pass != "" {
			pass = url.QueryEscape(pass)
		}
		dsn := ""
		if user != "" {
			dsn += user
			if pass != "" {
				dsn += ":" + pass
			}
			dsn += "@"
		}
		dsn += fmt.Sprintf("tcp(%s:%s)", host, port)
		if db != "" {
			dsn += "/" + db
		}
		return dsn
	case "redis":
		return fmt.Sprintf("redis://%s:%s", host, port)
	case "mongo":
		user := env["MONGO_INITDB_ROOT_USERNAME"]
		pass := env["MONGO_INITDB_ROOT_PASSWORD"]
		dbName := env["MONGO_INITDB_DATABASE"]
		if dbName == "" {
			dbName = "admin"
		}
		if user != "" && pass != "" {
			return fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", user, pass, host, port, dbName)
		}
		return fmt.Sprintf("mongodb://%s:%s/%s", host, port, dbName)
	}
	return ""
}
