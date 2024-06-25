package types

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserStore interface {
  // Create user attempts to create a new user for the given email and returns their uniquely assigned Id.
  // If a user with the given email already exists, their user Id should be returned.
  CreateUser(context context.Context, user *UserCreate) (*uuid.UUID, error)
}

type UserCreate struct {
  Email string `json:"email" db:"email" validate:"required,email,max=255"`
  Name string `json:"name" db:"name" validate:"required,min=2,max=64"`
  PreferredName *string `json:"preferred_name" db:"preferred_name" validate:"omitempty,min=2,max=64"`
}

type User struct {
  UserCreate
  Id uuid.UUID `json:"id" db:"id"`
  CreatedAt time.Time `json:"created_at" db:"created_at"`
}
