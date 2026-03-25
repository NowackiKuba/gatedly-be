package billing

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v84"
	stripePortal "github.com/stripe/stripe-go/v84/billingportal/session"
	stripeSession "github.com/stripe/stripe-go/v84/checkout/session"
)

// Service wraps Stripe operations used by the billing flow.
type Service interface {
	// CreateCheckoutSession creates a hosted Checkout Session and returns the
	// session URL to redirect the customer to.
	CreateCheckoutSession(ctx context.Context, params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error)

	// CreatePortalSession creates a Billing Portal Session so an existing
	// customer can manage their subscription.
	CreatePortalSession(ctx context.Context, params *stripe.BillingPortalSessionParams) (*stripe.BillingPortalSession, error)
}

type service struct {
	apiKey string
}

// New returns a Service configured with the given Stripe secret key.
func New(apiKey string) Service {
	return &service{apiKey: apiKey}
}

func (s *service) CreateCheckoutSession(
	ctx context.Context,
	params *stripe.CheckoutSessionParams,
) (*stripe.CheckoutSession, error) {
	stripe.Key = s.apiKey

	sess, err := stripeSession.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: create checkout session: %w", err)
	}
	return sess, nil
}

func (s *service) CreatePortalSession(
	ctx context.Context,
	params *stripe.BillingPortalSessionParams,
) (*stripe.BillingPortalSession, error) {
	stripe.Key = s.apiKey

	sess, err := stripePortal.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: create portal session: %w", err)
	}
	return sess, nil
}
