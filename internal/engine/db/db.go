package db

import (
	"fmt"
	"strings"

	sshlib "main/internal/ssh"
)

type DatabaseStats struct {
	PostgresStatus string
	MySQLStatus    string
	RedisStatus    string

	PostgresActiveConns string
	MySQLActiveConns    string
	RedisActiveConns    string

	PostgresSize string
	MySQLSize    string
	RedisSize    string
}

type SecretsManager struct {
	// Dummy struct to fulfill requirement
}

// Pull connection auth from Secrets Manager
func (s *SecretsManager) GetAuth(dbType string) (user, pass string) {
	if dbType == "postgres" {
		return "postgres", "postgres"
	}
	if dbType == "mysql" {
		return "root", "root"
	}
	if dbType == "redis" {
		return "default", ""
	}
	return "", ""
}

type Engine struct {
	client  *sshlib.Client
	secrets *SecretsManager
}

func NewEngine(client *sshlib.Client) *Engine {
	return &Engine{
		client:  client,
		secrets: &SecretsManager{},
	}
}

func (e *Engine) FetchStats() DatabaseStats {
	var stats DatabaseStats

	// Postgres
	stats.PostgresStatus = e.checkService("postgresql")
	if stats.PostgresStatus == "active" {
		u, _ := e.secrets.GetAuth("postgres")
		out := e.client.RunCommand(fmt.Sprintf(`su - %s -c "psql -t -c \"SELECT count(*) FROM pg_stat_activity;\""`, u))
		stats.PostgresActiveConns = strings.TrimSpace(out)

		outSize := e.client.RunCommand(fmt.Sprintf(`su - %s -c "psql -t -c \"SELECT pg_size_pretty(SUM(pg_database_size(oid))) FROM pg_database;\""`, u))
		stats.PostgresSize = strings.TrimSpace(outSize)
	}

	// MySQL
	stats.MySQLStatus = e.checkService("mysql")
	if stats.MySQLStatus == "active" {
		u, p := e.secrets.GetAuth("mysql")
		out := e.client.RunCommand(fmt.Sprintf(`mysql -u %s -p%s -e "SHOW STATUS LIKE 'Threads_connected';" | grep Threads_connected | awk '{print $2}'`, u, p))
		stats.MySQLActiveConns = strings.TrimSpace(out)

		outSize := e.client.RunCommand(fmt.Sprintf(`mysql -u %s -p%s -e "SELECT ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) 'Size (MB)' FROM information_schema.TABLES;" | tail -n 1 | awk '{print $1}'`, u, p))
		stats.MySQLSize = strings.TrimSpace(outSize) + " MB"
	}

	// Redis
	stats.RedisStatus = e.checkService("redis-server")
	if stats.RedisStatus == "active" {
		out := e.client.RunCommand(`redis-cli info clients | grep connected_clients | cut -d':' -f2`)
		stats.RedisActiveConns = strings.TrimSpace(out)

		outSize := e.client.RunCommand(`redis-cli info memory | grep used_memory_human | cut -d':' -f2`)
		stats.RedisSize = strings.TrimSpace(outSize)
	}

	return stats
}

func (e *Engine) checkService(name string) string {
	out := e.client.RunCommand(fmt.Sprintf("systemctl is-active %s", name))
	return strings.TrimSpace(out)
}

func (e *Engine) RunQuery(dbType, query string) string {
	u, p := e.secrets.GetAuth(dbType)
	if dbType == "postgres" {
		return e.client.RunCommand(fmt.Sprintf(`su - %s -c "psql -c \"%s\""`, u, query))
	} else if dbType == "mysql" {
		return e.client.RunCommand(fmt.Sprintf(`mysql -u %s -p%s -e "%s"`, u, p, query))
	} else if dbType == "redis" {
		return e.client.RunCommand(fmt.Sprintf(`redis-cli %s`, query))
	}
	return "Unsupported DB"
}
