package main

import (
	"os"
	// The embedded IANA time zone database resolves time zones on minimal base images without tzdata.
	_ "time/tzdata"

	"github.com/barthjs/photopi/internal/cli"
)

// Version is stamped into the binary at build time.
var Version = "dev"

// @title						PhotoPi
// @license.name				MIT
// @BasePath					/api
//
// @securityDefinitions.apikey	SessionAuth
// @in							cookie
// @name						session
//
//nolint:godot // The general API info for swag ends with an annotation.
func main() {
	os.Exit(cli.Execute(Version))
}
