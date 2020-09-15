// Package psql interfaces with Postgresql.
package psql

import (
	"context"
	"os"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type config struct {
	User     string
	Pass     string
	Host     string
	Port     string
	Database string
	Appname  string
}

// GetConfig parses the Postgresql connection settings from the environment.
//
// DEPRECATED: we just use the Postgresql env vars directly.
func GetConfig() (*pgx.ConnConfig, error) {
	config, err := pgx.ParseConfig("")
	if err != nil {
		return nil, err
	}
	// pass pgconn struct
	addAppname(&config.Config)
	return config, nil
}

// addAppname adds the application name to the Postgresql connection parameters.
func addAppname(c *pgconn.Config) {
	if name, ok := c.RuntimeParams["application_name"]; !ok || name == "" {
		c.RuntimeParams["application_name"] = "app"
	}
}

// ConfigFromEnv returns a database pool configured from default psql env vars.
// See: https://www.postgresql.org/docs/current/libpq-envars.html
//
// DEPRECATED: we just use the Postgresql env vars directly.
func ConfigFromEnv() *config {
	return &config{
		User:     os.Getenv("PGUSER"),
		Pass:     os.Getenv("PGPASS"),
		Host:     os.Getenv("PGHOST"),
		Port:     os.Getenv("PGPORT"),
		Database: os.Getenv("PGDATABASE"),
		Appname:  os.Getenv("PGAPPNAME"),
	}
}

// NewConnection makes a new connection to Postgresql using default PG* environment variables from the environment.
// See also NewConnectionFromApp if you want to override the application name in the connection settings.
func NewConnection(ctx context.Context) (*pgx.Conn, error) {
	return pgx.Connect(ctx, "")
}

// NewConnectionFromApp makes a new Postgresql connection using default PG* environment variables.
// It will add (potentially overriding) the provided application name to the connection settings to facilitate debugging.
// Prefer this function for helper tools that make direct database connections.
func NewConnectionFromApp(ctx context.Context, app string) (*pgx.Conn, error) {
	config, err := pgx.ParseConfig("")
	if err != nil {
		return nil, err
	}
	config.RuntimeParams["application_name"] = app
	return pgx.ConnectConfig(context.Background(), config)
}

// NewPool starts a new Postgresql pool using connection parameters defined in the environment.
// Prefer this pool function for the main database backend.
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, err
	}
	// pass pgconn struct
	//addAppname(&config.ConnConfig.Config)
	return pgxpool.ConnectConfig(context.Background(), config)
}
