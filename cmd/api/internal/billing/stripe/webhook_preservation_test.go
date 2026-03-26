package billing

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/cmd/api/internal/subscription"
	"toggly.com/m/pkg/logger"
)

// genUUID generates random UUIDs for property-based testing
func genUUID() gopter.Gen {
	return func(genParams *gopter.GenParameters) *gopter.GenResult {
		return gopter.NewGenResult(uuid.New(), gopter.NoShrinker)
	}
}

// TestPreservation_SubscriptionCreated verifies that customer.subscription.created
// events create subscription records with correct fields.
//
// **Validates: Requirements 3.1, 3.2, 3.3**
func TestPreservation_SubscriptionCreated(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("subscription.created creates record with correct fields", prop.ForAll(
		func(userID uuid.UUID, stripeSubID, stripeCustomerID, stripeProdID string, amount int64, status string) bool {
			// Setup
			webhookSecret := "whsec_test_secret"
			mockSubSvc := &captureSubscriptionService{subs: make(map[string]*domain.Subscription)}
			mockPacketRepo := &mockPacketRepository{}
			log := logger.New("test")
			wh := NewWebhookHandler(webhookSecret, mockSubSvc, mockPacketRepo, log)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(middleware.CORS(), middleware.Logger(log), middleware.Recoverer(log))
			v1 := router.Group("/api/v1")
			RegisterWebhookRoute(v1, wh)

			// Generate webhook payload
			timestamp := time.Now().Unix()
			payload := fmt.Sprintf(`{
				"id": "evt_test_%s",
				"object": "event",
				"api_version": "2026-02-25.clover",
				"type": "customer.subscription.created",
				"data": {
					"object": {
						"id": "%s",
						"object": "subscription",
						"customer": "%s",
						"status": "%s",
						"start_date": %d,
						"billing_cycle_anchor": %d,
						"cancel_at_period_end": false,
						"currency": "usd",
						"items": {
							"data": [{
								"price": {
									"unit_amount": %d,
									"product": "%s"
								}
							}]
						},
						"metadata": {
							"user_id": "%s"
						}
					}
				}
			}`, stripeSubID, stripeSubID, stripeCustomerID, status, timestamp, timestamp+2592000, amount, stripeProdID, userID.String())

			signature := generateStripeSignature(payload, webhookSecret, timestamp)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Stripe-Signature", signature)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify subscription was created with correct fields
			if w.Code != http.StatusOK {
				t.Logf("Expected HTTP 200, got %d: %s", w.Code, w.Body.String())
				return false
			}

			created := mockSubSvc.subs[stripeSubID]
			if created == nil {
				t.Logf("Subscription not created")
				return false
			}

			// Verify all critical fields
			mockPacket, _ := mockPacketRepo.GetByStripeProductID(context.Background(), stripeProdID)
			return created.UserID == userID &&
				created.StripeID == stripeSubID &&
				created.StripeCustomerID == stripeCustomerID &&
				created.Status == status &&
				created.Amount == int(amount) &&
				created.Currency == "usd" &&
				created.PacketID == mockPacket.ID &&
				created.Tier == mockPacket.Tier
		},
		genUUID(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Int64Range(0, 100000),
		gen.OneConstOf("active", "trialing", "past_due"),
	))

	properties.TestingRun(t)
}

// TestPreservation_SubscriptionUpdated verifies that customer.subscription.updated
// events update subscription records correctly.
//
// **Validates: Requirements 3.1, 3.2, 3.3**
func TestPreservation_SubscriptionUpdated(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("subscription.updated updates record correctly", prop.ForAll(
		func(stripeSubID string, newAmount int64, newStatus string, cancelAtEnd bool) bool {
			// Setup
			webhookSecret := "whsec_test_secret"
			existingUserID := uuid.New()
			existingSub := &domain.Subscription{
				Base:             domain.Base{ID: uuid.New()},
				UserID:           existingUserID,
				PacketID:         uuid.New(),
				Tier:             domain.BillingTierFree,
				Amount:           1000,
				Status:           "active",
				StripeID:         stripeSubID,
				StripeCustomerID: "cus_existing",
			}

			mockSubSvc := &captureSubscriptionService{
				subs: map[string]*domain.Subscription{stripeSubID: existingSub},
			}
			mockPacketRepo := &mockPacketRepository{}
			log := logger.New("test")
			wh := NewWebhookHandler(webhookSecret, mockSubSvc, mockPacketRepo, log)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(middleware.CORS(), middleware.Logger(log), middleware.Recoverer(log))
			v1 := router.Group("/api/v1")
			RegisterWebhookRoute(v1, wh)

			// Generate webhook payload
			timestamp := time.Now().Unix()
			payload := fmt.Sprintf(`{
				"id": "evt_test_update_%s",
				"object": "event",
				"api_version": "2026-02-25.clover",
				"type": "customer.subscription.updated",
				"data": {
					"object": {
						"id": "%s",
						"object": "subscription",
						"customer": "cus_existing",
						"status": "%s",
						"start_date": %d,
						"billing_cycle_anchor": %d,
						"cancel_at_period_end": %t,
						"currency": "usd",
						"items": {
							"data": [{
								"price": {
									"unit_amount": %d,
									"product": "prod_test123"
								}
							}]
						}
					}
				}
			}`, stripeSubID, stripeSubID, newStatus, timestamp, timestamp+2592000, cancelAtEnd, newAmount)

			signature := generateStripeSignature(payload, webhookSecret, timestamp)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Stripe-Signature", signature)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify subscription was updated
			if w.Code != http.StatusOK {
				t.Logf("Expected HTTP 200, got %d: %s", w.Code, w.Body.String())
				return false
			}

			updated := mockSubSvc.subs[stripeSubID]
			if updated == nil {
				t.Logf("Subscription not found after update")
				return false
			}

			// Verify updated fields
			mockPacket, _ := mockPacketRepo.GetByStripeProductID(context.Background(), "prod_test123")
			return updated.Amount == int(newAmount) &&
				updated.Status == newStatus &&
				*updated.CancelAtCurrentPeriodEnd == cancelAtEnd &&
				updated.PacketID == mockPacket.ID &&
				updated.Tier == mockPacket.Tier
		},
		gen.Identifier(),
		gen.Int64Range(0, 100000),
		gen.OneConstOf("active", "past_due", "canceled", "trialing"),
		gen.Bool(),
	))

	properties.TestingRun(t)
}

// TestPreservation_SubscriptionDeleted verifies that customer.subscription.deleted
// events set subscription status to "canceled".
//
// **Validates: Requirements 3.1, 3.2, 3.3**
func TestPreservation_SubscriptionDeleted(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("subscription.deleted sets status to canceled", prop.ForAll(
		func(stripeSubID string) bool {
			// Setup
			webhookSecret := "whsec_test_secret"
			existingSub := &domain.Subscription{
				Base:             domain.Base{ID: uuid.New()},
				UserID:           uuid.New(),
				PacketID:         uuid.New(),
				Tier:             domain.BillingTierPro,
				Status:           "active",
				StripeID:         stripeSubID,
				StripeCustomerID: "cus_test",
			}

			mockSubSvc := &captureSubscriptionService{
				subs: map[string]*domain.Subscription{stripeSubID: existingSub},
			}
			mockPacketRepo := &mockPacketRepository{}
			log := logger.New("test")
			wh := NewWebhookHandler(webhookSecret, mockSubSvc, mockPacketRepo, log)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(middleware.CORS(), middleware.Logger(log), middleware.Recoverer(log))
			v1 := router.Group("/api/v1")
			RegisterWebhookRoute(v1, wh)

			// Generate webhook payload
			timestamp := time.Now().Unix()
			canceledAt := timestamp - 3600
			payload := fmt.Sprintf(`{
				"id": "evt_test_delete_%s",
				"object": "event",
				"api_version": "2026-02-25.clover",
				"type": "customer.subscription.deleted",
				"data": {
					"object": {
						"id": "%s",
						"object": "subscription",
						"customer": "cus_test",
						"status": "canceled",
						"canceled_at": %d
					}
				}
			}`, stripeSubID, stripeSubID, canceledAt)

			signature := generateStripeSignature(payload, webhookSecret, timestamp)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Stripe-Signature", signature)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify subscription status is canceled
			if w.Code != http.StatusOK {
				t.Logf("Expected HTTP 200, got %d: %s", w.Code, w.Body.String())
				return false
			}

			updated := mockSubSvc.subs[stripeSubID]
			if updated == nil {
				t.Logf("Subscription not found after delete")
				return false
			}

			return updated.Status == "canceled" && updated.CanceledAt != nil
		},
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// TestPreservation_InvoicePaymentSucceeded verifies that invoice.payment_succeeded
// events reset evaluation counters and set status to "active".
//
// **Validates: Requirements 3.1, 3.2, 3.3**
func TestPreservation_InvoicePaymentSucceeded(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("invoice.payment_succeeded resets counters and sets active", prop.ForAll(
		func(stripeSubID string, previousEvaluations int) bool {
			// Setup
			webhookSecret := "whsec_test_secret"
			existingSub := &domain.Subscription{
				Base:             domain.Base{ID: uuid.New()},
				UserID:           uuid.New(),
				PacketID:         uuid.New(),
				Tier:             domain.BillingTierPro,
				Status:           "past_due",
				StripeID:         stripeSubID,
				StripeCustomerID: "cus_test",
				EvaluationsUsed:  previousEvaluations,
			}

			mockSubSvc := &captureSubscriptionService{
				subs: map[string]*domain.Subscription{stripeSubID: existingSub},
			}
			mockPacketRepo := &mockPacketRepository{}
			log := logger.New("test")
			wh := NewWebhookHandler(webhookSecret, mockSubSvc, mockPacketRepo, log)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(middleware.CORS(), middleware.Logger(log), middleware.Recoverer(log))
			v1 := router.Group("/api/v1")
			RegisterWebhookRoute(v1, wh)

			// Generate webhook payload
			timestamp := time.Now().Unix()
			payload := fmt.Sprintf(`{
				"id": "evt_test_invoice_%s",
				"object": "event",
				"api_version": "2026-02-25.clover",
				"type": "invoice.payment_succeeded",
				"data": {
					"object": {
						"id": "in_test123",
						"object": "invoice",
						"parent": {
							"subscription_details": {
								"subscription": {
									"id": "%s"
								}
							}
						}
					}
				}
			}`, stripeSubID, stripeSubID)

			signature := generateStripeSignature(payload, webhookSecret, timestamp)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Stripe-Signature", signature)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify evaluations reset and status is active
			if w.Code != http.StatusOK {
				t.Logf("Expected HTTP 200, got %d: %s", w.Code, w.Body.String())
				return false
			}

			updated := mockSubSvc.subs[stripeSubID]
			if updated == nil {
				t.Logf("Subscription not found after invoice payment")
				return false
			}

			return updated.Status == "active" &&
				updated.EvaluationsUsed == 0 &&
				updated.EvaluationsResetAt != nil
		},
		gen.Identifier(),
		gen.IntRange(0, 10000),
	))

	properties.TestingRun(t)
}

// TestPreservation_InvoicePaymentFailed verifies that invoice.payment_failed
// events set subscription status to "past_due".
//
// **Validates: Requirements 3.1, 3.2, 3.3**
func TestPreservation_InvoicePaymentFailed(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("invoice.payment_failed sets status to past_due", prop.ForAll(
		func(stripeSubID string) bool {
			// Setup
			webhookSecret := "whsec_test_secret"
			existingSub := &domain.Subscription{
				Base:             domain.Base{ID: uuid.New()},
				UserID:           uuid.New(),
				PacketID:         uuid.New(),
				Tier:             domain.BillingTierPro,
				Status:           "active",
				StripeID:         stripeSubID,
				StripeCustomerID: "cus_test",
			}

			mockSubSvc := &captureSubscriptionService{
				subs: map[string]*domain.Subscription{stripeSubID: existingSub},
			}
			mockPacketRepo := &mockPacketRepository{}
			log := logger.New("test")
			wh := NewWebhookHandler(webhookSecret, mockSubSvc, mockPacketRepo, log)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(middleware.CORS(), middleware.Logger(log), middleware.Recoverer(log))
			v1 := router.Group("/api/v1")
			RegisterWebhookRoute(v1, wh)

			// Generate webhook payload
			timestamp := time.Now().Unix()
			payload := fmt.Sprintf(`{
				"id": "evt_test_failed_%s",
				"object": "event",
				"api_version": "2026-02-25.clover",
				"type": "invoice.payment_failed",
				"data": {
					"object": {
						"id": "in_test_failed",
						"object": "invoice",
						"parent": {
							"subscription_details": {
								"subscription": {
									"id": "%s"
								}
							}
						}
					}
				}
			}`, stripeSubID, stripeSubID)

			signature := generateStripeSignature(payload, webhookSecret, timestamp)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Stripe-Signature", signature)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify status is past_due
			if w.Code != http.StatusOK {
				t.Logf("Expected HTTP 200, got %d: %s", w.Code, w.Body.String())
				return false
			}

			updated := mockSubSvc.subs[stripeSubID]
			if updated == nil {
				t.Logf("Subscription not found after payment failure")
				return false
			}

			return updated.Status == "past_due"
		},
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// TestPreservation_InvalidSignatureRejection verifies that webhooks with
// tampered signatures are rejected with HTTP 400.
//
// **Validates: Requirements 3.2**
func TestPreservation_InvalidSignatureRejection(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("invalid signature returns HTTP 400", prop.ForAll(
		func(stripeSubID string, tamperedSig string) bool {
			// Setup
			webhookSecret := "whsec_test_secret"
			mockSubSvc := &captureSubscriptionService{subs: make(map[string]*domain.Subscription)}
			mockPacketRepo := &mockPacketRepository{}
			log := logger.New("test")
			wh := NewWebhookHandler(webhookSecret, mockSubSvc, mockPacketRepo, log)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(middleware.CORS(), middleware.Logger(log), middleware.Recoverer(log))
			v1 := router.Group("/api/v1")
			RegisterWebhookRoute(v1, wh)

			// Generate webhook payload with INVALID signature
			timestamp := time.Now().Unix()
			payload := fmt.Sprintf(`{
				"id": "evt_test_%s",
				"object": "event",
				"api_version": "2026-02-25.clover",
				"type": "customer.subscription.created",
				"data": {
					"object": {
						"id": "%s",
						"object": "subscription",
						"customer": "cus_test",
						"status": "active"
					}
				}
			}`, stripeSubID, stripeSubID)

			// Use tampered signature instead of valid one
			invalidSignature := fmt.Sprintf("t=%d,v1=%s", timestamp, tamperedSig)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Stripe-Signature", invalidSignature)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify invalid signature is rejected with HTTP 400
			return w.Code == http.StatusBadRequest
		},
		gen.Identifier(),
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// captureSubscriptionService is a mock that captures subscription operations
type captureSubscriptionService struct {
	subscription.Service
	subs map[string]*domain.Subscription
}

func (m *captureSubscriptionService) Create(ctx context.Context, sub *domain.Subscription) error {
	m.subs[sub.StripeID] = sub
	return nil
}

func (m *captureSubscriptionService) Update(ctx context.Context, sub *domain.Subscription) error {
	m.subs[sub.StripeID] = sub
	return nil
}

func (m *captureSubscriptionService) GetByStripeID(ctx context.Context, stripeID string) (*domain.Subscription, error) {
	sub, ok := m.subs[stripeID]
	if !ok {
		return nil, fmt.Errorf("subscription not found")
	}
	return sub, nil
}

func (m *captureSubscriptionService) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	for _, sub := range m.subs {
		if sub.UserID == userID {
			return sub, nil
		}
	}
	return nil, fmt.Errorf("subscription not found")
}

func (m *captureSubscriptionService) Delete(ctx context.Context, id uuid.UUID) error {
	for key, sub := range m.subs {
		if sub.ID == id {
			delete(m.subs, key)
			return nil
		}
	}
	return fmt.Errorf("subscription not found")
}

func (m *captureSubscriptionService) GetAll(ctx context.Context) ([]*domain.Subscription, error) {
	subs := make([]*domain.Subscription, 0, len(m.subs))
	for _, sub := range m.subs {
		subs = append(subs, sub)
	}
	return subs, nil
}

func (m *captureSubscriptionService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	for _, sub := range m.subs {
		if sub.ID == id {
			return sub, nil
		}
	}
	return nil, fmt.Errorf("subscription not found")
}

func (m *captureSubscriptionService) IncrementEvaluations(ctx context.Context, userID uuid.UUID, count int) error {
	for _, sub := range m.subs {
		if sub.UserID == userID {
			sub.EvaluationsUsed += count
			return nil
		}
	}
	return fmt.Errorf("subscription not found")
}

func (m *captureSubscriptionService) GetEvaluationsUsed(ctx context.Context, userID uuid.UUID) (int, error) {
	for _, sub := range m.subs {
		if sub.UserID == userID {
			return sub.EvaluationsUsed, nil
		}
	}
	return 0, fmt.Errorf("subscription not found")
}

func (m *captureSubscriptionService) ResetEvaluations(ctx context.Context, userID uuid.UUID) error {
	for _, sub := range m.subs {
		if sub.UserID == userID {
			sub.EvaluationsUsed = 0
			now := time.Now()
			sub.EvaluationsResetAt = &now
			return nil
		}
	}
	return fmt.Errorf("subscription not found")
}

func (m *captureSubscriptionService) GetSubscriptionLimits(ctx context.Context, userID uuid.UUID) (int, int, error) {
	return 0, 0, nil
}

func (m *captureSubscriptionService) CheckEvaluationLimit(ctx context.Context, userID uuid.UUID) (bool, error) {
	return true, nil
}

func (m *captureSubscriptionService) GetSubscriptionStatus(ctx context.Context, userID uuid.UUID) (string, error) {
	for _, sub := range m.subs {
		if sub.UserID == userID {
			return sub.Status, nil
		}
	}
	return "", fmt.Errorf("subscription not found")
}

func (m *captureSubscriptionService) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	for _, sub := range m.subs {
		if sub.UserID == userID {
			sub.Status = "canceled"
			now := time.Now()
			sub.CanceledAt = &now
			return nil
		}
	}
	return fmt.Errorf("subscription not found")
}

func (m *captureSubscriptionService) ReactivateSubscription(ctx context.Context, userID uuid.UUID) error {
	for _, sub := range m.subs {
		if sub.UserID == userID {
			sub.Status = "active"
			return nil
		}
	}
	return fmt.Errorf("subscription not found")
}

func (m *captureSubscriptionService) UpdatePaymentMethod(ctx context.Context, userID uuid.UUID, paymentMethodID string) error {
	return nil
}

func (m *captureSubscriptionService) GetPaymentMethod(ctx context.Context, userID uuid.UUID) (string, error) {
	return "", nil
}

func (m *captureSubscriptionService) GetInvoices(ctx context.Context, userID uuid.UUID) ([]interface{}, error) {
	return nil, nil
}

func (m *captureSubscriptionService) GetUpcomingInvoice(ctx context.Context, userID uuid.UUID) (interface{}, error) {
	return nil, nil
}

func (m *captureSubscriptionService) UpdateSubscription(ctx context.Context, userID uuid.UUID, packetID uuid.UUID) error {
	return nil
}

func (m *captureSubscriptionService) GetSubscriptionHistory(ctx context.Context, userID uuid.UUID) ([]interface{}, error) {
	return nil, nil
}

func (m *captureSubscriptionService) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	for _, sub := range m.subs {
		if sub.ID == id {
			return sub, nil
		}
	}
	return nil, fmt.Errorf("subscription not found")
}

func (m *captureSubscriptionService) GetSubscriptionByStripeID(ctx context.Context, stripeID string) (*domain.Subscription, error) {
	return m.GetByStripeID(ctx, stripeID)
}

func (m *captureSubscriptionService) GetSubscriptionByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	return m.GetByUserID(ctx, userID)
}

func (m *captureSubscriptionService) CreateSubscription(ctx context.Context, sub *domain.Subscription) error {
	return m.Create(ctx, sub)
}

func (m *captureSubscriptionService) UpdateSubscriptionRecord(ctx context.Context, sub *domain.Subscription) error {
	return m.Update(ctx, sub)
}

func (m *captureSubscriptionService) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	return m.Delete(ctx, id)
}

func (m *captureSubscriptionService) GetAllSubscriptions(ctx context.Context) ([]*domain.Subscription, error) {
	return m.GetAll(ctx)
}
