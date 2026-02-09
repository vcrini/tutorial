package utils

import "github.com/google/uuid"

type UUIDService struct{}

func (UUIDService) New() string {
	return uuid.NewString()
}
