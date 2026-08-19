package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash of the given plaintext password.
// Passwords must never be stored as plain text.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword returns true if the plaintext password matches the hash.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
