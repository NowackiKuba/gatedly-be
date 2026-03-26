# Bugfix Requirements Document

## Introduction

The Stripe webhook endpoint at `/api/v1/webhooks/stripe` is failing signature verification with the error "webhook had no valid signature". The webhook secret is correctly configured (`whsec_RzvxgvmGbLMekkPaYHIl7jTw9MvSqhdH`), the Stripe-Signature header is present, and the code uses `webhook.ConstructEventIgnoringTolerance` to bypass timestamp validation. However, all webhook events from Stripe are being rejected.

The root cause is that the webhook handler is registered through Gin's router with middleware that may consume the request body before the webhook handler can read it. Stripe's signature verification requires the exact raw request body bytes, and if the body has been read by middleware or Gin's context handling, the signature verification will fail.

## Bug Analysis

### Current Behavior (Defect)

1.1 WHEN a Stripe webhook event is received at `/api/v1/webhooks/stripe` THEN the system fails signature verification with "webhook had no valid signature" error

1.2 WHEN the webhook handler attempts to read the request body with `io.ReadAll(r.Body)` THEN the body may already be consumed by Gin middleware, resulting in empty or incomplete payload for signature verification

1.3 WHEN signature verification fails THEN the webhook returns HTTP 400 "invalid signature" and Stripe webhook events are not processed

### Expected Behavior (Correct)

2.1 WHEN a Stripe webhook event is received at `/api/v1/webhooks/stripe` THEN the system SHALL successfully verify the signature using the raw request body bytes

2.2 WHEN the webhook handler reads the request body THEN it SHALL access the complete, unmodified raw body bytes that Stripe sent

2.3 WHEN signature verification succeeds THEN the webhook SHALL process the event and return HTTP 200

### Unchanged Behavior (Regression Prevention)

3.1 WHEN webhook events are successfully verified THEN the system SHALL CONTINUE TO process subscription.created, subscription.updated, subscription.deleted, invoice.payment_succeeded, and invoice.payment_failed events correctly

3.2 WHEN webhook signature verification fails due to an actually invalid signature THEN the system SHALL CONTINUE TO reject the request with HTTP 400

3.3 WHEN the webhook handler processes valid events THEN it SHALL CONTINUE TO update subscription records, reset evaluation counters, and handle payment status changes as before
