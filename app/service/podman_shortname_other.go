//go:build !darwin

package service

import (
	"context"
	"errors"
)

func RepairPodmanShortNameOnDarwin(ctx context.Context) (string, error) {
	_ = ctx
	return "", errors.New("unsupported platform")
}
