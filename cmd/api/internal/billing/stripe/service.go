package billing

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v84"
	stripePortal "github.com/stripe/stripe-go/v84/billingportal/session"
	stripeSession "github.com/stripe/stripe-go/v84/checkout/session"
	stripeCustomer "github.com/stripe/stripe-go/v84/customer"
	stripeInvoice "github.com/stripe/stripe-go/v84/invoice"
	stripeSub "github.com/stripe/stripe-go/v84/subscription"
)

// Service wraps Stripe operations used by the billing flow.
type Service interface {
	CreateCheckoutSession(ctx context.Context, params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error)
	CreatePortalSession(ctx context.Context, params *stripe.BillingPortalSessionParams) (*stripe.BillingPortalSession, error)
	GetCustomer(ctx context.Context, customerID string) (*stripe.Customer, error)
	GetDefaultPaymentMethod(ctx context.Context, customerID string) (*stripe.PaymentMethod, error)
	ListInvoices(ctx context.Context, customerID string, limit int64) ([]*stripe.Invoice, error)
	CancelSubscription(ctx context.Context, stripeSubID string) error
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

func (s *service) GetCustomer(ctx context.Context, customerID string) (*stripe.Customer, error) {
	stripe.Key = s.apiKey

	c, err := stripeCustomer.Get(customerID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe: get customer: %w", err)
	}
	return c, nil
}

func (s *service) GetDefaultPaymentMethod(ctx context.Context, customerID string) (*stripe.PaymentMethod, error) {
	stripe.Key = s.apiKey

	c, err := stripeCustomer.Get(customerID, &stripe.CustomerParams{
		Params: stripe.Params{
			Expand: []*string{stripe.String("invoice_settings.default_payment_method")},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("stripe: get customer payment method: %w", err)
	}
	if c.InvoiceSettings == nil || c.InvoiceSettings.DefaultPaymentMethod == nil {
		return nil, nil
	}
	return c.InvoiceSettings.DefaultPaymentMethod, nil
}

func (s *service) ListInvoices(ctx context.Context, customerID string, limit int64) ([]*stripe.Invoice, error) {
	stripe.Key = s.apiKey

	params := &stripe.InvoiceListParams{
		Customer: stripe.String(customerID),
	}
	params.Limit = &limit

	iter := stripeInvoice.List(params)
	var invoices []*stripe.Invoice
	for iter.Next() {
		invoices = append(invoices, iter.Invoice())
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("stripe: list invoices: %w", err)
	}
	return invoices, nil
}

func (s *service) CancelSubscription(ctx context.Context, stripeSubID string) error {
	stripe.Key = s.apiKey

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	if _, err := stripeSub.Update(stripeSubID, params); err != nil {
		return fmt.Errorf("stripe: cancel subscription: %w", err)
	}
	return nil
}
