package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvaluateRechargeExpression(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{name: "constant", expression: "1.25", want: 1.25},
		{name: "live rate", expression: "%fx_rate%", want: 7.2},
		{name: "balance conversion", expression: "1 / %fx_rate%", want: 1.0 / 7.2},
		{name: "parentheses", expression: "(%fx_rate% + 0.2) * 0.5", want: 3.7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluateRechargeExpression(tt.expression, 7.2)
			require.NoError(t, err)
			require.InDelta(t, tt.want, got, 1e-12)
		})
	}
}

func TestEvaluateRechargeExpressionRejectsInvalidInput(t *testing.T) {
	for _, expression := range []string{"", "1 / 0", "abc", "(1 + 2"} {
		_, err := evaluateRechargeExpression(expression, 7.2)
		require.Error(t, err, expression)
	}
}
