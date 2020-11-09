package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wvh/urn/internal/version"
)

const (
	// name of this application, also used as user-agent
	appName = "urnctl"

	// API requires authentication
	mustAuth = false

	// configuration environment variables
	envAPIServer = "URN_API_SERVER"
	envAPIToken  = "URN_API_TOKEN"
)

func run(args []string) error {
	var (
		flags = flag.NewFlagSet(args[0], flag.ExitOnError)

		//verbose    = flags.Bool("v", false, "verbose logging")
		//format     = flags.String("f", "Hi %s", "greeting format")
		apiServer   = flags.String("server", os.Getenv(envAPIServer), "API server URL, env: "+envAPIServer)
		apiToken    = flags.String("token", os.Getenv(envAPIToken), "API token, env: "+envAPIToken)
		showVersion = flags.Bool("version", false, "show client version")
	)
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	if *showVersion {
		fmt.Fprintf(os.Stderr, "%s %s (%s)\n", appName, version.Version, func() string {
			if version.Hash == "" {
				return "dev"
			}
			return fmt.Sprintf("commit: %s, branch: %s", version.Hash, version.Branch)
		}())
		return nil
	}

	if *apiServer == "" {
		return fmt.Errorf("API URL unset")
	}

	if mustAuth && *apiToken == "" {
		return fmt.Errorf("API token unset")
	}

	fmt.Fprintln(os.Stderr, appName, version.Version, "not doing anything!")

	api, err := NewAPIClient(appName, *apiServer, "abc123")
	if err != nil {
		return err
	}
	_ = api
	return nil
}

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", appName, err)
		os.Exit(1)
	}
}
