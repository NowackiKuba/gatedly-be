package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"toggly.com/m/cmd/api/internal/domain"
	"toggly.com/m/cmd/api/internal/middleware"
	"toggly.com/m/cmd/api/internal/packet"
	"toggly.com/m/cmd/api/internal/subscription"
	"toggly.com/m/pkg/logger"
)

// TestBugCondition_WebhookSignatureVerificationFails demonstrates the bug where
// Stripe webhook signature verification fails because Gin middleware consumes
// the request body before the webhook handler can read it.
//
// **CRITICAL**: This test MUST FAIL on unfixed code - failure confirms the bug exists.
// **DO NOT attempt to fix the test or the code when it fails**.
// **NOTE**: This test encodes the expected behavior - it will validate the fix when it passes.
//
// **Validates: Requirements 1.1, 1.2, 1.3**
func TestBugCondition_WebhookSignatureVerificationFails(t *testing.T) {
	// Test webhook secret (matches the one in the bug report)
	webhookSecret := "whsec_RzvxgvmGbLMekkPaYHIl7jTw9MvSqhdH"

	// Mock services
	mockSubSvc := &mockSubscriptionService{}
	mockPacketRepo := &mockPacketRepository{}
	log := logger.New("test")

	// Create webhook handler
	wh := NewWebhookHandler(webhookSecret, mockSubSvc, mockPacketRepo, log)

	// Set up Gin router with middleware (simulating production setup)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.CORS(), middleware.Logger(log), middleware.Recoverer(log))

	v1 := router.Group("/api/v1")
	RegisterWebhookRoute(v1, wh)

	// Test cases: valid Stripe webhook events that should succeed
	testCases := []struct {
		name      string
		eventType string
		payload   string
	}{
		{
			name:      "customer.subscription.created",
			eventType: "customer.subscription.created",
			payload: `{
				"id": "evt_test_subscription_created",
				"object": "event",
				"api_version": "2026-02-25.clover",
				"type": "customer.subscription.created",
				"data": {
					"object": {
						"id": "sub_test123",
						"object": "subscription",
						"customer": "cus_test123",
						"status": "active",
						"start_date": 1234567890,
						"billing_cycle_anchor": 1234567890,
						"cancel_at_period_end": false,
						"currency": "usd",
						"items": {
							"data": [{
								"price": {
									"unit_amount": 1000,
									"product": "prod_test123"
								}
							}]
						},
						"metadata": {
							"user_id": "550e8400-e29b-41d4-a716-446655440000"
						}
					}
				}
			}`,
		},
		{
			name:      "invoice.payment_succeeded",
			eventType: "invoice.payment_succeeded",
			payload: `{
				"id": "evt_test_invoice_succeeded",
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
									"id": "sub_test123"
								}
							}
						}
					}
				}
			}`,
		},
		{
			name:      "customer.subscription.updated",
			eventType: "customer.subscription.updated",
			payload: `{
				"id": "evt_test_subscription_updated",
				"object": "event",
				"api_version": "2026-02-25.clover",
				"type": "customer.subscription.updated",
				"data": {
					"object": {
						"id": "sub_test123",
						"object": "subscription",
						"customer": "cus_test123",
						"status": "active",
						"start_date": 1234567890,
						"billing_cycle_anchor": 1234567890,
						"cancel_at_period_end": false,
						"currency": "usd",
						"items": {
							"data": [{
								"price": {
									"unit_amount": 2000,
									"product": "prod_test456"
								}
							}]
						}
					}
				}
			}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate valid Stripe signature
			timestamp := time.Now().Unix()
			signature := generateStripeSignature(tc.payload, webhookSecret, timestamp)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Stripe-Signature", signature)

			// Record response
			w := httptest.NewRecorder()

			// Execute request through Gin router (with middleware)
			router.ServeHTTP(w, req)

			// EXPECTED BEHAVIOR: Signature verification should succeed and return HTTP 200
			// ON UNFIXED CODE: This will FAIL with HTTP 400 "invalid signature" because
			// Gin middleware consumes the request body before the handler can read it
			if w.Code != http.StatusOK {
				t.Errorf("COUNTEREXAMPLE FOUND: %s webhook failed signature verification\n"+
					"Expected: HTTP 200 (signature verification succeeds)\n"+
					"Got: HTTP %d\n"+
					"Response: %s\n"+
					"This confirms the bug exists: request body consumed before signature verification",
					tc.eventType, w.Code, w.Body.String())
			}
		})
	}
}

// generateStripeSignature creates a valid Stripe-Signature header value
// using the same algorithm Stripe uses (HMAC SHA256)
func generateStripeSignature(payload, secret string, timestamp int64) string {
	// Stripe signature format: t=timestamp,v1=signature
	signedPayload := fmt.Sprintf("%d.%s", timestamp, payload)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("t=%s,v1=%s", strconv.FormatInt(timestamp, 10), signature)
}

// Mock implementations for testing

type mockSubscriptionService struct {
	subscription.Service
}

func (m *mockSubscriptionService) Create(ctx context.Context, sub *domain.Subscription) error {
	return nil
}

func (m *mockSubscriptionService) Update(ctx context.Context, sub *domain.Subscription) error {
	return nil
}

func (m *mockSubscriptionService) GetByStripeID(ctx context.Context, stripeID string) (*domain.Subscription, error) {
	// Return a mock subscription
	return &domain.Subscription{
		Base: domain.Base{
			ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		UserID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		PacketID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Tier:     domain.BillingTierPro,
		Interval: domain.BillingIntervalFree,
		Status:   "active",
		StripeID: stripeID,
	}, nil
}

type mockPacketRepository struct {
	packet.Repository
}

func (m *mockPacketRepository) GetByStripeProductID(ctx context.Context, productID string) (*domain.Packet, error) {
	// Return a mock packet
	return &domain.Packet{
		Base: domain.Base{
			ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		},
		Name:            "Pro Plan",
		Tier:            domain.BillingTierPro,
		Interval:        domain.BillingIntervalFree,
		StripeProductID: productID,
	}, nil
}
