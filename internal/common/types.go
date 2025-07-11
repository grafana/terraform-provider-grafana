// Package common provides common utilities for the Grafana Terraform provider.
// Test comment to trigger schema update workflow
package common

import (
	"errors"
	"math"
)

func ToInt32[T ~int | ~int64](val T) (int32, error) {
	if val < math.MinInt32 || val > math.MaxInt32 {
		return 0, errors.New("value overflows int32")
	}
	return int32(val), nil
}
