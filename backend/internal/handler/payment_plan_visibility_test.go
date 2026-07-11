package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestVisibleCheckoutGroupInfoRedactsDisabledFields(t *testing.T) {
	limit := 10.0
	info := service.PlanGroupInfo{
		RateMultiplier:     2,
		PeakRateEnabled:    true,
		PeakStart:          "09:00",
		PeakEnd:            "18:00",
		PeakRateMultiplier: 3,
		DailyLimitUSD:      &limit,
		WeeklyLimitUSD:     &limit,
		MonthlyLimitUSD:    &limit,
		ModelScopes:        []string{"claude"},
	}

	visible := visibleCheckoutGroupInfo(&service.PaymentConfig{}, info)
	require.Nil(t, visible.RateMultiplier)
	require.Nil(t, visible.PeakRateEnabled)
	require.Empty(t, visible.PeakStart)
	require.Nil(t, visible.DailyLimitUSD)
	require.Nil(t, visible.WeeklyLimitUSD)
	require.Nil(t, visible.MonthlyLimitUSD)
	require.Nil(t, visible.ModelScopes)
}

func TestVisibleCheckoutGroupInfoIncludesEnabledFields(t *testing.T) {
	limit := 10.0
	info := service.PlanGroupInfo{RateMultiplier: 2, DailyLimitUSD: &limit}
	cfg := &service.PaymentConfig{SubscriptionShowRate: true, SubscriptionShow5hLimit: true}

	visible := visibleCheckoutGroupInfo(cfg, info)
	require.NotNil(t, visible.RateMultiplier)
	require.Equal(t, 2.0, *visible.RateMultiplier)
	require.Equal(t, &limit, visible.DailyLimitUSD)
}
