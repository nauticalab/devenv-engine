package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/nauticalab/devenv-engine/internal/manager/auth"
	"github.com/nauticalab/devenv-engine/internal/manager/client"
)

// AuthRunList lists the current authentication information
func AuthRunList(managerURL string) {
	if managerURL == "" {
		fmt.Fprintf(os.Stderr, "Error: manager URL is required. Set DEVEN_MANAGER_URL env var or configure in ~/.devenv/config.yaml\n")
		os.Exit(1)
	}

	// Create manager client
	authProvider := auth.NewK8sSAProvider(nil, "", "", "")
	c := client.NewClient(managerURL, authProvider)

	// Get identity
	whoami, err := c.WhoAmI(context.Background())
	if err != nil {
		fmt.Printf("Error getting authentication info: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Authenticated as: %s\n", whoami.Username)
	fmt.Printf("Type: %s\n", whoami.Type)
	fmt.Printf("Developer: %s\n", whoami.Developer)
	if whoami.Namespace != "" {
		fmt.Printf("Namespace: %s\n", whoami.Namespace)
	}
}
