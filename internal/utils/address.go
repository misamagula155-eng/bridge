package utils

import (
	"encoding/hex"
	"errors"
	"strings"
)

var (
	ErrInvalidPublicAddressMissing = errors.New("public address must be set")
	ErrInvalidPublicAddressFormat  = errors.New("public address must be a valid hex string")
	ErrInvalidPublicAddressLength  = errors.New("public address must be 64 characters long")
)

type PublicAddress string

func NewPublicAddressFromString(a string) (PublicAddress, error) {
	if err := PublicAddress(strings.TrimSpace(a)).Validate(); err != nil {
		return "", err
	}
	return PublicAddress(a), nil
}

func (a PublicAddress) String() string {
	return string(a)
}

func (a PublicAddress) Validate() error {
	if a == "" {
		return ErrInvalidPublicAddressMissing
	}

	if len(a) != 64 {
		return ErrInvalidPublicAddressLength
	}

	if _, err := hex.DecodeString(string(a)); err != nil {
		return ErrInvalidPublicAddressFormat
	}

	return nil
}
