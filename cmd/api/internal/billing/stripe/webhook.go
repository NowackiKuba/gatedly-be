package billing

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/cmd/api/internal/packet"
	"toggly.com/m/cmd/api/internal/subscription"
)

// WebhookHandler handles inbound Stripe webhook events.
type WebhookHandler struct {
	webhookSecret string
	subSvc        subscription.Service
	packetRepo    packet.Repository
	log           *slog.Logger
}

// NewWebhookHandler returns a new WebhookHandler.
func NewWebhookHandler(
	webhookSecret string,
	subSvc subscription.Service,
	packetRepo packet.Repository,
	log *slog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		webhookSecret: webhookSecret,
		subSvc:        subSvc,
		packetRepo:    packetRepo,
		log:           log,
	}
}

// Handle is the HTTP handler for POST /webhooks/stripe.
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("webhook: read body", "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sig, h.webhookSecret)
	if err != nil {
		h.log.Error("webhook: signature verification failed", "err", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	h.log.Info("webhook: received", "type", event.Type, "id", event.ID)

	switch event.Type {
	case "customer.subscription.created":
		err = h.handleSubscriptionCreated(r, event)
	case "customer.subscription.updated":
		err = h.handleSubscriptionUpdated(r, event)
	case "customer.subscription.deleted":
		err = h.handleSubscriptionDeleted(r, event)
	case "invoice.payment_succeeded":
		err = h.handleInvoicePaymentSucceeded(r, event)
	case "invoice.payment_failed":
		err = h.handleInvoicePaymentFailed(r, event)
	default:
		h.log.Info("webhook: unhandled event type", "type", event.Type)
	}

	if err != nil {
		h.log.Error("webhook: handler error", "type", event.Type, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleSubscriptionCreated(r *http.Request, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	packetID, tier, interval, err := h.resolvePacket(r, &sub)
	if err != nil {
		return err
	}

	userIDStr, ok := sub.Metadata["user_id"]
	if !ok || userIDStr == "" {
		return fmt.Errorf("webhook: subscription.created missing user_id metadata")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("webhook: invalid user_id in metadata: %w", err)
	}

	// In v84 there are no CurrentPeriodStart/End fields on Subscription.
	// Use StartDate as period start and derive end from the billing cycle anchor.
	periodStart := time.Unix(sub.StartDate, 0)
	periodEnd := time.Unix(sub.BillingCycleAnchor, 0)
	cancelAtEnd := sub.CancelAtPeriodEnd
	now := time.Now()

	isTrial := sub.Status == "trialing"
	var trialStart, trialEnd *time.Time
	if sub.TrialStart != 0 {
		t := time.Unix(sub.TrialStart, 0)
		trialStart = &t
	}
	if sub.TrialEnd != 0 {
		t := time.Unix(sub.TrialEnd, 0)
		trialEnd = &t
	}

	amount := 0
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		amount = int(sub.Items.Data[0].Price.UnitAmount)
	}

	newSub := &domain.Subscription{
		UserID:                   userID,
		PacketID:                 packetID,
		Tier:                     tier,
		Amount:                   amount,
		Currency:                 string(sub.Currency),
		Interval:                 interval,
		Status:                   string(sub.Status),
		IsTrial:                  isTrial,
		TrialStartsAt:            trialStart,
		TrialEndsAt:              trialEnd,
		CurrentPeriodStartedAt:   periodStart,
		CurrentPeriodEndsAt:      periodEnd,
		CancelAtCurrentPeriodEnd: &cancelAtEnd,
		EvaluationsResetAt:       &now,
		StripeID:                 sub.ID,
		StripeCustomerID:         sub.Customer.ID,
	}

	if err := h.subSvc.Create(r.Context(), newSub); err != nil {
		return fmt.Errorf("webhook: create subscription: %w", err)
	}
	return nil
}

func (h *WebhookHandler) handleSubscriptionUpdated(r *http.Request, event stripe.Event) error {
	var stripeSub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &stripeSub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	existing, err := h.subSvc.GetByStripeID(r.Context(), stripeSub.ID)
	if err != nil {
		return fmt.Errorf("webhook: get subscription: %w", err)
	}

	packetID, tier, interval, err := h.resolvePacket(r, &stripeSub)
	if err != nil {
		return err
	}

	cancelAtEnd := stripeSub.CancelAtPeriodEnd

	existing.PacketID = packetID
	existing.Tier = tier
	existing.Interval = interval
	existing.Status = string(stripeSub.Status)
	existing.Currency = string(stripeSub.Currency)
	existing.CurrentPeriodEndsAt = time.Unix(stripeSub.BillingCycleAnchor, 0)
	existing.CancelAtCurrentPeriodEnd = &cancelAtEnd
	existing.IsTrial = stripeSub.Status == "trialing"

	if stripeSub.Items != nil && len(stripeSub.Items.Data) > 0 {
		existing.Amount = int(stripeSub.Items.Data[0].Price.UnitAmount)
	}
	if stripeSub.TrialStart != 0 {
		t := time.Unix(stripeSub.TrialStart, 0)
		existing.TrialStartsAt = &t
	}
	if stripeSub.TrialEnd != 0 {
		t := time.Unix(stripeSub.TrialEnd, 0)
		existing.TrialEndsAt = &t
	}
	if stripeSub.CanceledAt != 0 {
		t := time.Unix(stripeSub.CanceledAt, 0)
		existing.CanceledAt = &t
	}

	if err := h.subSvc.Update(r.Context(), existing); err != nil {
		return fmt.Errorf("webhook: update subscription: %w", err)
	}
	return nil
}

func (h *WebhookHandler) handleSubscriptionDeleted(r *http.Request, event stripe.Event) error {
	var stripeSub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &stripeSub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	existing, err := h.subSvc.GetByStripeID(r.Context(), stripeSub.ID)
	if err != nil {
		return fmt.Errorf("webhook: get subscription: %w", err)
	}

	existing.Status = "canceled"
	if stripeSub.CanceledAt != 0 {
		t := time.Unix(stripeSub.CanceledAt, 0)
		existing.CanceledAt = &t
	}

	if err := h.subSvc.Update(r.Context(), existing); err != nil {
		return fmt.Errorf("webhook: cancel subscription: %w", err)
	}
	return nil
}

func (h *WebhookHandler) handleInvoicePaymentSucceeded(r *http.Request, event stripe.Event) error {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}
	if inv.Parent == nil || inv.Parent.SubscriptionDetails == nil || inv.Parent.SubscriptionDetails.Subscription == nil {
		return nil // one-time payment, nothing to do
	}

	existing, err := h.subSvc.GetByStripeID(r.Context(), inv.Parent.SubscriptionDetails.Subscription.ID)
	if err != nil {
		return fmt.Errorf("webhook: get subscription: %w", err)
	}

	now := time.Now()
	existing.Status = "active"
	existing.EvaluationsUsed = 0
	existing.EvaluationsResetAt = &now

	if err := h.subSvc.Update(r.Context(), existing); err != nil {
		return fmt.Errorf("webhook: reset evaluations: %w", err)
	}
	return nil
}

func (h *WebhookHandler) handleInvoicePaymentFailed(r *http.Request, event stripe.Event) error {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}
	if inv.Parent == nil || inv.Parent.SubscriptionDetails == nil || inv.Parent.SubscriptionDetails.Subscription == nil {
		return nil
	}

	existing, err := h.subSvc.GetByStripeID(r.Context(), inv.Parent.SubscriptionDetails.Subscription.ID)
	if err != nil {
		return fmt.Errorf("webhook: get subscription: %w", err)
	}

	existing.Status = "past_due"
	if err := h.subSvc.Update(r.Context(), existing); err != nil {
		return fmt.Errorf("webhook: mark past_due: %w", err)
	}
	return nil
}

// resolvePacket looks up the local Packet from the Stripe subscription's first line item product.
func (h *WebhookHandler) resolvePacket(
	r *http.Request,
	sub *stripe.Subscription,
) (uuid.UUID, domain.BillingTier, domain.BillingInterval, error) {
	if sub.Items == nil || len(sub.Items.Data) == 0 {
		return uuid.Nil, "", "", fmt.Errorf("webhook: subscription has no line items")
	}

	productID := sub.Items.Data[0].Price.Product.ID
	p, err := h.packetRepo.GetByStripeProductID(r.Context(), productID)
	if err != nil {
		return uuid.Nil, "", "", fmt.Errorf("webhook: lookup packet by product %s: %w", productID, err)
	}
	if p == nil {
		return uuid.Nil, "", "", fmt.Errorf("webhook: no packet found for stripe product %s", productID)
	}

	return p.ID, p.Tier, p.Interval, nil
}
