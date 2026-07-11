package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultFXRateFallback         = 7.2
	defaultSubscriptionExpression = "1"
	fxRatePlaceholder             = "%fx_rate%"
)

var fxRateCache struct {
	sync.Mutex
	rate      float64
	expiresAt time.Time
}

func (s *PaymentConfigService) ApplyEffectiveRechargeExpressions(ctx context.Context, cfg *PaymentConfig) {
	if cfg == nil {
		return
	}
	fxRate := cfg.FXRateFallback
	if fxRate <= 0 || math.IsNaN(fxRate) || math.IsInf(fxRate, 0) {
		fxRate = defaultFXRateFallback
	}
	if strings.Contains(cfg.BalanceRechargeExpression, fxRatePlaceholder) || strings.Contains(cfg.SubscriptionRechargeExpression, fxRatePlaceholder) {
		if liveRate, err := fetchUSDToCNYRate(ctx); err == nil {
			fxRate = liveRate
		}
	}
	if value, err := evaluateRechargeExpression(cfg.BalanceRechargeExpression, fxRate); err == nil && value > 0 {
		cfg.BalanceRechargeMultiplier = value
	}
	if value, err := evaluateRechargeExpression(cfg.SubscriptionRechargeExpression, fxRate); err == nil && value > 0 {
		cfg.SubscriptionUSDToCNYRate = value
	}
}

func fetchUSDToCNYRate(ctx context.Context) (float64, error) {
	now := time.Now()
	fxRateCache.Lock()
	if fxRateCache.rate > 0 && now.Before(fxRateCache.expiresAt) {
		rate := fxRateCache.rate
		fxRateCache.Unlock()
		return rate, nil
	}
	fxRateCache.Unlock()

	requestCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, "https://open.er-api.com/v6/latest/USD", nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("exchange rate endpoint returned %s", resp.Status)
	}
	var payload struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	rate := payload.Rates["CNY"]
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return 0, fmt.Errorf("invalid CNY exchange rate")
	}
	fxRateCache.Lock()
	fxRateCache.rate = rate
	fxRateCache.expiresAt = now.Add(6 * time.Hour)
	fxRateCache.Unlock()
	return rate, nil
}

func evaluateRechargeExpression(expression string, fxRate float64) (float64, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return 0, fmt.Errorf("expression is empty")
	}
	expression = strings.ReplaceAll(expression, fxRatePlaceholder, strconv.FormatFloat(fxRate, 'g', -1, 64))
	p := rechargeExpressionParser{input: expression}
	value, err := p.parseExpression()
	if err != nil {
		return 0, err
	}
	p.skipSpaces()
	if p.pos != len(p.input) || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("invalid expression")
	}
	return value, nil
}

type rechargeExpressionParser struct {
	input string
	pos   int
}

func (p *rechargeExpressionParser) parseExpression() (float64, error) {
	value, err := p.parseTerm()
	for err == nil {
		p.skipSpaces()
		if p.pos >= len(p.input) || (p.input[p.pos] != '+' && p.input[p.pos] != '-') {
			break
		}
		op := p.input[p.pos]
		p.pos++
		var rhs float64
		rhs, err = p.parseTerm()
		if op == '+' {
			value += rhs
		} else {
			value -= rhs
		}
	}
	return value, err
}

func (p *rechargeExpressionParser) parseTerm() (float64, error) {
	value, err := p.parseFactor()
	for err == nil {
		p.skipSpaces()
		if p.pos >= len(p.input) || (p.input[p.pos] != '*' && p.input[p.pos] != '/') {
			break
		}
		op := p.input[p.pos]
		p.pos++
		var rhs float64
		rhs, err = p.parseFactor()
		if op == '*' {
			value *= rhs
		} else if rhs == 0 {
			return 0, fmt.Errorf("division by zero")
		} else {
			value /= rhs
		}
	}
	return value, err
}

func (p *rechargeExpressionParser) parseFactor() (float64, error) {
	p.skipSpaces()
	if p.pos >= len(p.input) {
		return 0, fmt.Errorf("unexpected end of expression")
	}
	if p.input[p.pos] == '+' || p.input[p.pos] == '-' {
		op := p.input[p.pos]
		p.pos++
		value, err := p.parseFactor()
		if op == '-' {
			value = -value
		}
		return value, err
	}
	if p.input[p.pos] == '(' {
		p.pos++
		value, err := p.parseExpression()
		p.skipSpaces()
		if err != nil || p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return 0, fmt.Errorf("unclosed parenthesis")
		}
		p.pos++
		return value, nil
	}
	start := p.pos
	for p.pos < len(p.input) && ((p.input[p.pos] >= '0' && p.input[p.pos] <= '9') || p.input[p.pos] == '.') {
		p.pos++
	}
	if start == p.pos {
		return 0, fmt.Errorf("unexpected token")
	}
	return strconv.ParseFloat(p.input[start:p.pos], 64)
}

func (p *rechargeExpressionParser) skipSpaces() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t') {
		p.pos++
	}
}
