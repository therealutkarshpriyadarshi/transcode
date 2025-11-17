package transcoder

import (
	"crypto/rand"
	"encoding/hex"
	"os"
)

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	bytes := make([]byte, length/2+1)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// removeFile removes a file, ignoring errors
func removeFile(path string) {
	os.Remove(path)
}

// readFile reads a file and returns its contents
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
