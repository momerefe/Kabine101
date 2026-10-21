package db

import "embed"

// Migrations are compiled into the CLI, avoiding working-directory dependencies.
//
//go:embed migrations/*.sql
var Migrations embed.FS

const SchemaVersion = 2
