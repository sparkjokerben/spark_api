package dto

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupMappers_RedactUserRatesAndPreserveAdminRates(t *testing.T) {
	group := &service.Group{
		ID:                           7,
		Name:                         "hidden-rate",
		RateMultiplier:               1.5,
		ImageRateIndependent:         true,
		ImageRateMultiplier:          1.6,
		BatchImageDiscountMultiplier: 0.8,
		BatchImageHoldMultiplier:     0.2,
		VideoRateIndependent:         true,
		VideoRateMultiplier:          1.7,
		PeakRateEnabled:              true,
		PeakStart:                    "09:00",
		PeakEnd:                      "18:00",
		PeakRateMultiplier:           2,
	}

	userJSON, err := json.Marshal(GroupFromService(group))
	require.NoError(t, err)
	for _, field := range []string{
		"rate_multiplier", "image_rate_independent", "image_rate_multiplier",
		"batch_image_discount_multiplier", "batch_image_hold_multiplier",
		"video_rate_independent", "video_rate_multiplier", "peak_rate_enabled",
		"peak_start", "peak_end", "peak_rate_multiplier",
	} {
		require.NotContains(t, string(userJSON), `"`+field+`"`)
	}

	adminJSON, err := json.Marshal(GroupFromServiceAdmin(group))
	require.NoError(t, err)
	for _, field := range []string{"rate_multiplier", "image_rate_multiplier", "video_rate_multiplier", "peak_rate_multiplier"} {
		require.Contains(t, string(adminJSON), `"`+field+`"`)
	}
}
