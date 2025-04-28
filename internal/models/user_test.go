package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUser_Success(t *testing.T) {
	user, err := NewUser(
		"John",
		"Doe",
		"johndoe",
		"securePassword123",
		"john.doe@example.com",
		"US",
	)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "John", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
	assert.Equal(t, "johndoe", user.Nickname)
	assert.Equal(t, "john.doe@example.com", user.Email)
	assert.Equal(t, "US", user.Country)
	assert.NotEmpty(t, user.Password) // password should be hashed
}

func TestNewUser_InvalidPassword(t *testing.T) {
	_, err := NewUser(
		"John",
		"Doe",
		"johndoe",
		"123", // too short
		"john.doe@example.com",
		"US",
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password must be at least 8 characters")
}

func TestUser_SanitizeForOutput(t *testing.T) {
	user := &User{
		Password: "somehashedpassword",
	}

	user.SanitizeForOutput()

	assert.Equal(t, "", user.Password)
}
