package google

import (
	"fmt"
	"strings"
)

func ParseSecretPath(secretPath string) error {
	splitSecretPath := strings.Split(secretPath, "/")

	if len(splitSecretPath) == 0 {
		return fmt.Errorf("secret path is empty")
	}

	if len(splitSecretPath) != 6 {
		return fmt.Errorf("secret path must be 6 characters long")
	}

	if !strings.HasPrefix(secretPath, "projects/") {
		return fmt.Errorf("secret path must start with 'projects/'")
	}

	if !strings.Contains(secretPath, "/secrets/") {
		return fmt.Errorf("secret path must contain '/secrets/'")
	}

	if !strings.Contains(secretPath, "/versions/") {
		return fmt.Errorf("secret path must contain '/versions/'")
	}

	return nil
}
