package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SubscriptionSummaryItem represents a subscription item in summary
type SubscriptionSummaryItem struct {
	ID              int64   `json:"id"`
	GroupID         int64   `json:"group_id"`
	GroupName       string  `json:"group_name"`
	Status          string  `json:"status"`
	DailyUsedUSD    float64 `json:"daily_used_usd,omitempty"`
	DailyLimitUSD   float64 `json:"daily_limit_usd,omitempty"`
	WeeklyUsedUSD   float64 `json:"weekly_used_usd,omitempty"`
	WeeklyLimitUSD  float64 `json:"weekly_limit_usd,omitempty"`
	MonthlyUsedUSD  float64 `json:"monthly_used_usd,omitempty"`
	MonthlyLimitUSD float64 `json:"monthly_limit_usd,omitempty"`
	ExpiresAt       *string `json:"expires_at,omitempty"`
}

// SubscriptionProgressInfo represents subscription with progress info
type SubscriptionProgressInfo struct {
	Subscription *dto.UserSubscription         `json:"subscription"`
	Progress     *service.SubscriptionProgress `json:"progress"`
}

// SubscriptionHandler handles user subscription operations
type SubscriptionHandler struct {
	subscriptionService *service.SubscriptionService
	configService       *service.PaymentConfigService
}

// NewSubscriptionHandler creates a new user subscription handler
func NewSubscriptionHandler(subscriptionService *service.SubscriptionService, configService *service.PaymentConfigService) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionService: subscriptionService,
		configService:       configService,
	}
}

func (h *SubscriptionHandler) displayConfig(c *gin.Context) service.SubscriptionDisplayConfig {
	if h.configService == nil {
		return service.SubscriptionDisplayConfig{ShowRate: true, ShowPeakRate: true, Show5hLimit: true, ShowWeekLimit: true, ShowMonthLimit: true, ShowModelScopes: true}
	}
	cfg, err := h.configService.GetSubscriptionDisplayConfig(c.Request.Context())
	if err != nil {
		return service.SubscriptionDisplayConfig{}
	}
	return cfg
}

func sanitizeUserSubscription(out *dto.UserSubscription, cfg service.SubscriptionDisplayConfig) {
	if out == nil {
		return
	}
	out.ShowRate = cfg.ShowRate
	out.ShowPeakRate = cfg.ShowPeakRate
	out.Show5hLimit = cfg.Show5hLimit
	out.ShowWeekLimit = cfg.ShowWeekLimit
	out.ShowMonthLimit = cfg.ShowMonthLimit
	if out.Group == nil {
		return
	}
	if !cfg.ShowRate {
		out.Group.RateMultiplier = 0
	}
	if !cfg.ShowPeakRate {
		out.Group.PeakRateEnabled = false
		out.Group.PeakStart = ""
		out.Group.PeakEnd = ""
		out.Group.PeakRateMultiplier = 0
	}
	if !cfg.Show5hLimit {
		out.Group.DailyLimitUSD = nil
		out.DailyUsageUSD = 0
		out.DailyWindowStart = nil
	}
	if !cfg.ShowWeekLimit {
		out.Group.WeeklyLimitUSD = nil
		out.WeeklyUsageUSD = 0
		out.WeeklyWindowStart = nil
	}
	if !cfg.ShowMonthLimit {
		out.Group.MonthlyLimitUSD = nil
		out.MonthlyUsageUSD = 0
		out.MonthlyWindowStart = nil
	}
}

// List handles listing current user's subscriptions
// GET /api/v1/subscriptions
func (h *SubscriptionHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	subscriptions, err := h.subscriptionService.ListUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.UserSubscription, 0, len(subscriptions))
	cfg := h.displayConfig(c)
	for i := range subscriptions {
		item := dto.UserSubscriptionFromService(&subscriptions[i])
		sanitizeUserSubscription(item, cfg)
		out = append(out, *item)
	}
	response.Success(c, out)
}

// GetActive handles getting current user's active subscriptions
// GET /api/v1/subscriptions/active
func (h *SubscriptionHandler) GetActive(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	subscriptions, err := h.subscriptionService.ListActiveUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.UserSubscription, 0, len(subscriptions))
	cfg := h.displayConfig(c)
	for i := range subscriptions {
		item := dto.UserSubscriptionFromService(&subscriptions[i])
		sanitizeUserSubscription(item, cfg)
		out = append(out, *item)
	}
	response.Success(c, out)
}

// GetProgress handles getting subscription progress for current user
// GET /api/v1/subscriptions/progress
func (h *SubscriptionHandler) GetProgress(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	// Get all active subscriptions with progress
	subscriptions, err := h.subscriptionService.ListActiveUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	result := make([]SubscriptionProgressInfo, 0, len(subscriptions))
	cfg := h.displayConfig(c)
	for i := range subscriptions {
		sub := &subscriptions[i]
		progress, err := h.subscriptionService.GetSubscriptionProgress(c.Request.Context(), sub.ID)
		if err != nil {
			// Skip subscriptions with errors
			continue
		}
		item := dto.UserSubscriptionFromService(sub)
		sanitizeUserSubscription(item, cfg)
		if !cfg.Show5hLimit {
			progress.Daily = nil
		}
		if !cfg.ShowWeekLimit {
			progress.Weekly = nil
		}
		if !cfg.ShowMonthLimit {
			progress.Monthly = nil
		}
		result = append(result, SubscriptionProgressInfo{
			Subscription: item,
			Progress:     progress,
		})
	}

	response.Success(c, result)
}

// GetSummary handles getting a summary of current user's subscription status
// GET /api/v1/subscriptions/summary
func (h *SubscriptionHandler) GetSummary(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	// Get all active subscriptions
	subscriptions, err := h.subscriptionService.ListActiveUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var totalUsed float64
	cfg := h.displayConfig(c)
	items := make([]SubscriptionSummaryItem, 0, len(subscriptions))

	for _, sub := range subscriptions {
		item := SubscriptionSummaryItem{
			ID:      sub.ID,
			GroupID: sub.GroupID,
			Status:  sub.Status,
		}
		if cfg.Show5hLimit {
			item.DailyUsedUSD = sub.DailyUsageUSD
		}
		if cfg.ShowWeekLimit {
			item.WeeklyUsedUSD = sub.WeeklyUsageUSD
		}
		if cfg.ShowMonthLimit {
			item.MonthlyUsedUSD = sub.MonthlyUsageUSD
		}

		// Add group info if preloaded
		if sub.Group != nil {
			item.GroupName = sub.Group.Name
			if cfg.Show5hLimit && sub.Group.DailyLimitUSD != nil {
				item.DailyLimitUSD = *sub.Group.DailyLimitUSD
			}
			if cfg.ShowWeekLimit && sub.Group.WeeklyLimitUSD != nil {
				item.WeeklyLimitUSD = *sub.Group.WeeklyLimitUSD
			}
			if cfg.ShowMonthLimit && sub.Group.MonthlyLimitUSD != nil {
				item.MonthlyLimitUSD = *sub.Group.MonthlyLimitUSD
			}
		}

		// Format expiration time
		if !sub.ExpiresAt.IsZero() {
			formatted := sub.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
			item.ExpiresAt = &formatted
		}

		// Track total usage (use monthly as the most comprehensive)
		if cfg.ShowMonthLimit {
			totalUsed += sub.MonthlyUsageUSD
		}

		items = append(items, item)
	}

	summary := struct {
		ActiveCount   int                       `json:"active_count"`
		TotalUsedUSD  float64                   `json:"total_used_usd"`
		Subscriptions []SubscriptionSummaryItem `json:"subscriptions"`
	}{
		ActiveCount:   len(subscriptions),
		TotalUsedUSD:  totalUsed,
		Subscriptions: items,
	}

	response.Success(c, summary)
}
