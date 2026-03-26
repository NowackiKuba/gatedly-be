# Stripe Webhook Signature Fix Design

## Overview

The Stripe webhook endpoint at `/api/v1/webhooks/stripe` is failing signature verification because Gin middleware consumes the request body before the webhook handler can read it. Stripe's signature verification requires the exact raw request body bytes that were sent. The fix involves bypassing Gin's context handling for this specific endpoint and reading the raw body directly from the HTTP request before any middleware processes it.

## Glossary

- **Bug_Condition (C)**: The condition that triggers the bug - when a Stripe webhook event is received and the request body has been consumed by Gin middleware before signature verification
- **Property (P)**: The desired behavior when webhook events are received - signature verification should succeed using the complete raw request body bytes
- **Preservation**: Existing webhook event processing logic (subscription.created, subscription.updated, etc.) that must remain unchanged by the fix
- **HandleKeyPress**: The webhook handler function in `cmd/api/internal/billing/stripe/webhook.go` that processes incoming Stripe webhook events
- **Gin Context**: The Gin framework's request context that may buffer or consume the request body through middleware processing
- **Raw Body**: The unmodified byte stream from the HTTP request that Stripe uses to compute the signature

## Bug Details

### Bug Condition

The bug manifests when a Stripe webhook event is received at `/api/v1/webhooks/stripe`. The webhook handler is registered through Gin's router with `g.POST("/webhooks/stripe", func(c *gin.Context) { wh.Handle(c.Writer, c.Request) })`, which means the request passes through Gin's middleware stack. Gin may buffer or consume the request body during context processing, and when the handler calls `io.ReadAll(r.Body)`, the body is empty or incomplete, causing signature verification to fail.

**Formal Specification:**
```
FUNCTION isBugCondition(input)
  INPUT: input of type HTTPRequest
  OUTPUT: boolean
  
  RETURN input.path == "/api/v1/webhooks/stripe"
         AND input.method == "POST"
         AND input.header["Stripe-Signature"] EXISTS
         AND requestBodyConsumedByMiddleware(input)
         AND signatureVerificationFails(input)
END FUNCTION
```

### Examples

- **Example 1**: Stripe sends a `customer.subscription.created` event with valid signature → Handler reads empty body → Signature verification fails with "webhook had no valid signature" → Returns HTTP 400
- **Example 2**: Stripe sends an `invoice.payment_succeeded` event with valid signature → Gin middleware buffers body → Handler gets incomplete payload → Signature verification fails → Returns HTTP 400
- **Example 3**: Stripe sends a `customer.subscription.updated` event → Body consumed before handler → Signature check fails → Subscription update not processed
- **Edge case**: Webhook with invalid signature → Should still return HTTP 400 (expected behavior preserved)

## Expected Behavior

### Preservation Requirements

**Unchanged Behaviors:**
- Webhook event processing logic for subscription.created, subscription.updated, subscription.deleted, invoice.payment_succeeded, and invoice.payment_failed must continue to work exactly as before
- Webhook signature verification must continue to reject requests with actually invalid signatures (HTTP 400)
- Subscription record updates, evaluation counter resets, and payment status changes must remain unchanged
- All other API endpoints using Gin middleware must continue to function normally

**Scope:**
All inputs that do NOT involve the Stripe webhook endpoint at `/api/v1/webhooks/stripe` should be completely unaffected by this fix. This includes:
- All other authenticated API endpoints
- Other billing endpoints (checkout, portal, usage, etc.)
- Gin middleware processing for non-webhook routes
- Request body handling for other POST endpoints

## Hypothesized Root Cause

Based on the bug description, the most likely issues are:

1. **Gin Context Body Buffering**: Gin's context may automatically read and buffer the request body when the handler is invoked through `c.Request`, making subsequent reads return empty data
   - The handler uses `io.ReadAll(r.Body)` which expects an unread body stream
   - Once read, `r.Body` cannot be read again without explicit restoration

2. **Middleware Body Consumption**: One of the middleware functions (CORS, Logger, Recoverer) may be reading the request body for logging or processing purposes

3. **HTTP MaxBytesReader Interaction**: The `http.MaxBytesReader` wrapper may interact poorly with Gin's body handling, causing the body to be consumed prematurely

4. **Request Body Not Restored**: If the body is read by middleware, it's not being restored using `c.Request.Body = io.NopCloser(bytes.NewBuffer(payload))` pattern

## Correctness Properties

Property 1: Bug Condition - Webhook Signature Verification Succeeds

_For any_ HTTP POST request to `/api/v1/webhooks/stripe` with a valid Stripe-Signature header and valid webhook payload, the fixed webhook handler SHALL successfully verify the signature using the complete raw request body bytes and process the webhook event, returning HTTP 200.

**Validates: Requirements 2.1, 2.2, 2.3**

Property 2: Preservation - Webhook Event Processing Logic

_For any_ successfully verified webhook event (after the fix), the webhook handler SHALL process subscription.created, subscription.updated, subscription.deleted, invoice.payment_succeeded, and invoice.payment_failed events exactly as the original code did, preserving all subscription updates, evaluation counter resets, and payment status changes.

**Validates: Requirements 3.1, 3.3**

## Fix Implementation

### Changes Required

Assuming our root cause analysis is correct:

**File**: `cmd/api/internal/billing/stripe/routes.go`

**Function**: `RegisterWebhookRoute`

**Specific Changes**:
1. **Bypass Gin Context for Body Reading**: Modify the webhook route registration to read the raw body before passing to the handler
   - Read the body bytes directly from `c.Request.Body` before calling `wh.Handle`
   - Store the raw bytes and restore them to `c.Request.Body` using `io.NopCloser(bytes.NewBuffer(payload))`

2. **Alternative Approach - Use Native HTTP Handler**: Register the webhook endpoint using Gin's `router.Any()` with a native `http.HandlerFunc` that bypasses Gin's context entirely
   - This ensures no middleware touches the request body
   - Signature verification gets the pristine raw body

3. **Pass Raw Body to Handler**: Modify `WebhookHandler.Handle` signature to accept pre-read body bytes if needed
   - Or ensure the body is properly restored before the handler reads it

4. **Preserve MaxBytesReader**: Keep the `http.MaxBytesReader` protection in the handler to prevent large payloads

5. **Maintain Error Handling**: Ensure all error responses (400, 500) continue to work correctly

**Recommended Implementation**: Read the body in the Gin wrapper function and restore it before calling the handler:

```go
func RegisterWebhookRoute(g *gin.RouterGroup, wh *WebhookHandler) {
	g.POST("/webhooks/stripe", func(c *gin.Context) {
		// Read body before Gin can consume it
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}
		// Restore body for the handler
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		wh.Handle(c.Writer, c.Request)
	})
}
```

## Testing Strategy

### Validation Approach

The testing strategy follows a two-phase approach: first, surface counterexamples that demonstrate the bug on unfixed code, then verify the fix works correctly and preserves existing behavior.

### Exploratory Bug Condition Checking

**Goal**: Surface counterexamples that demonstrate the bug BEFORE implementing the fix. Confirm or refute the root cause analysis. If we refute, we will need to re-hypothesize.

**Test Plan**: Write tests that simulate Stripe webhook POST requests with valid signatures to the `/api/v1/webhooks/stripe` endpoint. Run these tests on the UNFIXED code to observe signature verification failures and understand the root cause.

**Test Cases**:
1. **Subscription Created Test**: Send a valid `customer.subscription.created` webhook event with correct signature (will fail on unfixed code with "webhook had no valid signature")
2. **Invoice Payment Succeeded Test**: Send a valid `invoice.payment_succeeded` webhook event (will fail on unfixed code)
3. **Subscription Updated Test**: Send a valid `customer.subscription.updated` webhook event (will fail on unfixed code)
4. **Large Payload Test**: Send a webhook with payload near the 65536 byte limit (may fail on unfixed code, tests MaxBytesReader interaction)

**Expected Counterexamples**:
- Signature verification fails with "webhook had no valid signature" error
- Possible causes: empty body read, incomplete body read, body consumed by middleware

### Fix Checking

**Goal**: Verify that for all inputs where the bug condition holds, the fixed function produces the expected behavior.

**Pseudocode:**
```
FOR ALL input WHERE isBugCondition(input) DO
  result := handleWebhook_fixed(input)
  ASSERT signatureVerificationSucceeds(result)
  ASSERT result.statusCode == 200
END FOR
```

### Preservation Checking

**Goal**: Verify that for all inputs where the bug condition does NOT hold, the fixed function produces the same result as the original function.

**Pseudocode:**
```
FOR ALL input WHERE NOT isBugCondition(input) DO
  ASSERT handleWebhook_original(input) = handleWebhook_fixed(input)
END FOR
```

**Testing Approach**: Property-based testing is recommended for preservation checking because:
- It generates many test cases automatically across the input domain
- It catches edge cases that manual unit tests might miss
- It provides strong guarantees that behavior is unchanged for all non-buggy inputs

**Test Plan**: Observe behavior on UNFIXED code first for webhook event processing, then write property-based tests capturing that behavior.

**Test Cases**:
1. **Subscription Event Processing Preservation**: Verify that after signature verification succeeds, subscription.created events create subscription records with correct fields (user_id, packet_id, tier, amount, etc.)
2. **Invoice Event Processing Preservation**: Verify that invoice.payment_succeeded events reset evaluation counters and update status to "active"
3. **Invalid Signature Rejection Preservation**: Verify that webhooks with invalid signatures continue to return HTTP 400
4. **Other Endpoints Preservation**: Verify that other API endpoints (auth, billing, etc.) continue to work with Gin middleware

### Unit Tests

- Test webhook signature verification with valid Stripe signatures
- Test webhook signature verification with invalid signatures (should fail)
- Test edge cases (missing Stripe-Signature header, empty body, oversized payload)
- Test that each webhook event type (subscription.created, invoice.payment_succeeded, etc.) processes correctly after signature verification

### Property-Based Tests

- Generate random valid Stripe webhook payloads and verify signature verification succeeds
- Generate random subscription data and verify webhook processing creates/updates records correctly
- Test that all webhook event types preserve their processing logic across many scenarios

### Integration Tests

- Test full webhook flow: Stripe sends event → Signature verified → Event processed → Database updated → HTTP 200 returned
- Test webhook failure flow: Invalid signature → HTTP 400 returned → No database changes
- Test that webhook endpoint works independently of other API endpoints and middleware
