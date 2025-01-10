package services

import "fmt"

// User represents a user in the system.
type User struct {
	Email    string
	Password string
	Verified bool
}

// Mock database of users.
var users = []User{
	{Email: "test@example.com", Password: "password123", Verified: true},
	{Email: "unverified@example.com", Password: "password456", Verified: false},
}

// Authenticate checks if the user credentials are valid.
func Authenticate(email, password string) (*User, error) {
	for _, user := range users {
		if user.Email == email && user.Password == password {
			if !user.Verified {
				return nil, fmt.Errorf("email not verified")
			}
			return &user, nil
		}
	}
	return nil, fmt.Errorf("invalid email or password")
}

// GetUserByEmail retrieves a user by their email address.
func GetUserByEmail(email string) (*User, error) {
	for _, user := range users {
		if user.Email == email {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}
