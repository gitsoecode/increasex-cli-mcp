package auth

import "os"

func LoadEnvToken() string {
	return os.Getenv("INCREASE_API_KEY")
}
