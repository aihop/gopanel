//go:build !darwin
// +build !darwin

package service

import (
	"context"
	"fmt"
)

func podmanMachineRegistriesGet(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("unsupported platform")
}

func podmanMachineRegistriesSet(ctx context.Context, mirrors []string) error {
	return fmt.Errorf("unsupported platform")
}
