package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"user-microservice/internal/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type FilterOptions struct {
	Country   string
	Email     string
	Nickname  string
	FirstName string
	LastName  string
}

type PaginationOptions struct {
	Page     int
	PageSize int
}

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByNickname(ctx context.Context, nickname string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdatePassword(ctx context.Context, id, password string) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter FilterOptions, pagination PaginationOptions) ([]*models.User, int, error)
}

type HealthChecker interface {
	CheckHealth() error
}

type PostgresUserRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewPostgresUserRepository(db *sqlx.DB, logger *zap.Logger) *PostgresUserRepository {
	return &PostgresUserRepository{
		db:     db,
		logger: logger.With(zap.String("component", "postgres_repository")),
	}
}

// CheckHealth checks the health of the database connection
func (r *PostgresUserRepository) CheckHealth() error {
	return r.db.Ping()
}

// Create adds a new user
func (r *PostgresUserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, first_name, last_name, nickname, password, email, country, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	r.logger.Debug("creating user",
		zap.String("id", user.ID),
		zap.String("email", user.Email),
		zap.String("nickname", user.Nickname))

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.FirstName,
		user.LastName,
		user.Nickname,
		user.Password,
		user.Email,
		user.Country,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("error creating user", zap.Error(err))
		return errors.Wrap(err, "error inserting user into database")
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, first_name, last_name, nickname, email, country, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	r.logger.Debug("retrieving user by ID", zap.String("id", id))

	var user models.User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		r.logger.Error("error retrieving user by ID", zap.Error(err))
		return nil, errors.Wrap(err, "error retrieving user from database")
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, first_name, last_name, nickname, email, country, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	r.logger.Debug("retrieving user by email", zap.String("email", email))

	var user models.User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		r.logger.Error("error retrieving user by email", zap.Error(err))
		return nil, errors.Wrap(err, "error retrieving user from database")
	}

	return &user, nil
}

// GetByNickname retrieves a user by nickname
func (r *PostgresUserRepository) GetByNickname(ctx context.Context, nickname string) (*models.User, error) {
	query := `
		SELECT id, first_name, last_name, nickname, email, country, created_at, updated_at
		FROM users
		WHERE nickname = $1
	`

	r.logger.Debug("retrieving user by nickname", zap.String("nickname", nickname))

	var user models.User
	err := r.db.GetContext(ctx, &user, query, nickname)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		r.logger.Error("error retrieving user by nickname", zap.Error(err))
		return nil, errors.Wrap(err, "error retrieving user from database")
	}

	return &user, nil
}

// Update updates an existing user
func (r *PostgresUserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, nickname = $3, email = $4, country = $5, updated_at = $6
		WHERE id = $7
	`

	r.logger.Debug("updating user", zap.String("id", user.ID))

	user.UpdatedAt = time.Now().UTC()

	result, err := r.db.ExecContext(ctx, query,
		user.FirstName,
		user.LastName,
		user.Nickname,
		user.Email,
		user.Country,
		user.UpdatedAt,
		user.ID,
	)
	if err != nil {
		r.logger.Error("error updating user", zap.Error(err))
		return errors.Wrap(err, "error updating user in the database")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error checking affected rows")
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdatePassword updates a user's password
func (r *PostgresUserRepository) UpdatePassword(ctx context.Context, id, password string) error {
	query := `
		UPDATE users
		SET password = $1, updated_at = $2
		WHERE id = $3
	`

	r.logger.Debug("updating user password", zap.String("id", id))

	now := time.Now().UTC()

	result, err := r.db.ExecContext(ctx, query, password, now, id)
	if err != nil {
		r.logger.Error("error updating password", zap.Error(err))
		return errors.Wrap(err, "error updating password in the database")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error checking affected rows")
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Delete removes a user
func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	r.logger.Debug("removing user", zap.String("id", id))

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("error removing user", zap.Error(err))
		return errors.Wrap(err, "error removing user from the database")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error checking affected rows")
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// List returns a paginated list of users with filters
func (r *PostgresUserRepository) List(ctx context.Context, filter FilterOptions, pagination PaginationOptions) ([]*models.User, int, error) {
	// Build base query
	baseQuery := `
		SELECT id, first_name, last_name, nickname, email, country, created_at, updated_at
		FROM users
		WHERE 1=1
	`
	countQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE 1=1
	`

	// Add filters
	var conditions string
	var args []interface{}
	var argIndex int = 1

	if filter.Country != "" {
		conditions += fmt.Sprintf(" AND country ILIKE $%d", argIndex)
		args = append(args, filter.Country)
		argIndex++
	}

	if filter.Email != "" {
		conditions += fmt.Sprintf(" AND email ILIKE $%d", argIndex)
		args = append(args, filter.Email)
		argIndex++
	}

	if filter.Nickname != "" {
		conditions += fmt.Sprintf(" AND nickname ILIKE $%d", argIndex)
		args = append(args, filter.Nickname)
		argIndex++
	}

	if filter.FirstName != "" {
		conditions += fmt.Sprintf(" AND first_name ILIKE $%d", argIndex)
		args = append(args, filter.FirstName)
		argIndex++
	}

	if filter.LastName != "" {
		conditions += fmt.Sprintf(" AND last_name ILIKE $%d", argIndex)
		args = append(args, filter.LastName)
		argIndex++
	}

	// Add pagination
	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.PageSize < 1 {
		pagination.PageSize = 10
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	limit := pagination.PageSize

	// Final query
	query := baseQuery + conditions + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Count query
	countQueryFinal := countQuery + conditions

	r.logger.Debug("listing users",
		zap.Any("filter", filter),
		zap.Any("pagination", pagination))

	// Execute count query
	var total int
	err := r.db.GetContext(ctx, &total, countQueryFinal, args[:argIndex-1]...)
	if err != nil {
		r.logger.Error("error counting users", zap.Error(err))
		return nil, 0, errors.Wrap(err, "error counting users in the database")
	}

	// Execute main query
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("error listing users", zap.Error(err))
		return nil, 0, errors.Wrap(err, "error listing users in the database")
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.StructScan(&user); err != nil {
			r.logger.Error("error scanning user", zap.Error(err))
			return nil, 0, errors.Wrap(err, "error scanning user from the database")
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating over users", zap.Error(err))
		return nil, 0, errors.Wrap(err, "error iterating over users from the database")
	}

	return users, total, nil
}
