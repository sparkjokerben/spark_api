package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestVisibleCheckoutGroupInfoRedactsDisabledFields(t *testing.T) {
	limit := 10.0
	info := service.PlanGroupInfo{
		DailyLimitUSD:   &limit,
		WeeklyLimitUSD:  &limit,
		MonthlyLimitUSD: &limit,
		ModelScopes:     []string{"claude"},
	}

	visible := visibleCheckoutGroupInfo(&service.PaymentConfig{}, info)
	require.Nil(t, visible.DailyLimitUSD)
	require.Nil(t, visible.WeeklyLimitUSD)
	require.Nil(t, visible.MonthlyLimitUSD)
	require.Nil(t, visible.ModelScopes)
}

func TestPublicPaymentConfigRedactsRechargeAndDisplayRates(t *testing.T) {
	cfg := &service.PaymentConfig{
		BalanceRechargeMultiplier:      0.14,
		BalanceRechargeExpression:      "secret balance expression",
		SubscriptionRechargeExpression: "secret subscription expression",
		SubscriptionShowRate:           true,
		SubscriptionShowPeakRate:       true,
		StripePublishableKey:           "pk_test_public",
	}

	raw, err := json.Marshal(publicPaymentConfigFromService(cfg))
	require.NoError(t, err)
	require.Contains(t, string(raw), `"stripe_publishable_key":"pk_test_public"`)
	for _, field := range []string{
		"balance_recharge_multiplier", "balance_recharge_expression",
		"subscription_recharge_expression", "subscription_show_rate",
		"subscription_show_peak_rate",
	} {
		require.NotContains(t, string(raw), field)
	}
}

func TestVisibleCheckoutGroupInfoIncludesEnabledFields(t *testing.T) {
	limit := 10.0
	info := service.PlanGroupInfo{DailyLimitUSD: &limit}
	cfg := &service.PaymentConfig{SubscriptionShow5hLimit: true}

	visible := visibleCheckoutGroupInfo(cfg, info)
	require.Equal(t, &limit, visible.DailyLimitUSD)
}
