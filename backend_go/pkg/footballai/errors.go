package footballai

import "fmt"

// errConfig builds a configuration/validation error.
func errConfig(format string, args ...interface{}) error {
	return fmt.Errorf("footballai: "+format, args...)
}
