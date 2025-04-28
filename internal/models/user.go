package models

import (
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
// @Description User object representing the user in the system
// @model
type User struct {
	ID        string    `json:"id" db:"id"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	Nickname  string    `json:"nickname" db:"nickname"`
	Password  string    `json:"password,omitempty" db:"password"`
	Email     string    `json:"email" db:"email"`
	Country   string    `json:"country" db:"country"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func NewUser(firstName, lastName, nickname, password, email, country string) (*User, error) {
	tempUser := &User{
		FirstName: firstName,
		LastName:  lastName,
		Nickname:  nickname,
		Password:  password,
		Email:     email,
		Country:   country,
	}

	if err := tempUser.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid input data")
	}

	if err := tempUser.HashPassword(password); err != nil {
		return nil, errors.Wrap(err, "failed to hash password")
	}

	now := time.Now().UTC()

	tempUser.ID = uuid.New().String()
	tempUser.CreatedAt = now
	tempUser.UpdatedAt = now

	return tempUser, nil
}

// Validate validates the user data
func (u *User) Validate() error {
	if u.FirstName == "" {
		return errors.New("first name is required")
	}

	if u.LastName == "" {
		return errors.New("last name is required")
	}

	if u.Nickname == "" {
		return errors.New("nickname is required")
	}

	if u.Country == "" {
		return errors.New("country is required")
	}

	if u.Email == "" {
		return errors.New("email is required")
	}

	if err := u.ValidateEmail(u.Email); err != nil {
		return err
	}

	if err := u.ValidatePassword(u.Password); err != nil {
		return err
	}

	return nil
}

// UpdatePassword updates the user's password
func (u *User) UpdatePassword(newPassword string) error {
	if err := u.ValidatePassword(newPassword); err != nil {
		return err
	}

	if err := u.HashPassword(newPassword); err != nil {
		return err
	}

	u.UpdatedAt = time.Now().UTC()

	return nil
}

// Update updates the user's data
func (u *User) Update(firstName, lastName, nickname, email, country string) error {
	if firstName != "" {
		u.FirstName = firstName
	}

	if lastName != "" {
		u.LastName = lastName
	}

	if nickname != "" {
		u.Nickname = nickname
	}

	if email != "" {
		if err := u.ValidateEmail(email); err != nil {
			return err
		}
		u.Email = email
	}

	if country != "" {
		u.Country = country
	}

	u.UpdatedAt = time.Now().UTC()

	return nil
}

func (u *User) ValidateEmail(email string) error {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	match, err := regexp.MatchString(emailRegex, email)
	if err != nil {
		return errors.New("error validating email")
	}
	if !match {
		return errors.New("invalid email format")
	}
	return nil
}

func (u *User) SanitizeForOutput() {
	u.Password = ""
}

func (u *User) HashPassword(newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, "failed to generate password hash")
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) ValidatePassword(password string) error {
	if password == "" {
		return errors.New("password cannot be empty")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	return nil
}
