package source

import (
	"encoding/json"
	"os"
	"regexp"
)

// Matches a pattern like "$(ENV_VAR_NAME)"
var envRegex = regexp.MustCompile(`\$\(([a-zA-Z_-]+)\)`)

// ReplaceEnv replaces all environment variable placeholders with the contents from the
// environment.
func ReplaceEnv(value string) string {
	return envRegex.ReplaceAllStringFunc(value, func(matched string) string {
		return os.Getenv(envRegex.FindStringSubmatch(matched)[1])
	})
}

// Credential overrides JSON parsing to translate "$(ENV_VAR_NAME)" into the value of the
// environment variable.
type Credential string

func (s *Credential) UnmarshalJSON(data []byte) error {
	var plaintext string
	if err := json.Unmarshal(data, &plaintext); err != nil {
		return err
	}

	*s = (Credential(ReplaceEnv(plaintext)))

	return nil
}
