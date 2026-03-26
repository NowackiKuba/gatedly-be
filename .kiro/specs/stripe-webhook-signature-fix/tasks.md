# Implementation Plan

- [x] 1. Write bug condition exploration test
  - **Property 1: Bug Condition** - Webhook Signature Verification Fails Due to Body Consumption
  - **CRITICAL**: This test MUST FAIL on unfixed code - failure confirms the bug exists
  - **DO NOT attempt to fix the test or the code when it fails**
  - **NOTE**: This test encodes the expected behavior - it will validate the fix when it passes after implementation
  - **GOAL**: Surface counterexamples that demonstrate the bug exists
  - **Scoped PBT Approach**: Scope the property to concrete failing cases - valid Stripe webhook events with correct signatures sent to `/api/v1/webhooks/stripe`
  - Test that webhook signature verification fails when request body is consumed by Gin middleware before handler reads it
  - Test cases: `customer.subscription.created`, `invoice.payment_succeeded`, `customer.subscription.updated` events with valid Stripe-Signature headers
  - The test assertions should verify that signature verification succeeds (HTTP 200) for valid webhook events
  - Run test on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS with "webhook had no valid signature" error (this is correct - it proves the bug exists)
  - Document counterexamples found: which webhook events fail, what error messages appear, whether body is empty or incomplete
  - Mark task complete when test is written, run, and failure is documented
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 2. Write preservation property tests (BEFORE implementing fix)
  - **Property 2: Preservation** - Webhook Event Processing and Invalid Signature Rejection
  - **IMPORTANT**: Follow observation-first methodology
  - Observe behavior on UNFIXED code for non-buggy scenarios (if signature verification were to succeed)
  - Write property-based tests capturing webhook event processing logic:
    - For `customer.subscription.created`: verify subscription record created with correct user_id, packet_id, tier, amount, status fields
    - For `customer.subscription.updated`: verify subscription record updated with new packet_id, tier, status, cancel_at_period_end
    - For `customer.subscription.deleted`: verify subscription status set to "canceled"
    - For `invoice.payment_succeeded`: verify evaluation counters reset to 0 and status set to "active"
    - For `invoice.payment_failed`: verify subscription status set to "past_due"
  - Write test for invalid signature rejection: webhook with tampered signature should return HTTP 400
  - Property-based testing generates many test cases for stronger guarantees
  - Note: These tests may not run successfully on UNFIXED code due to signature verification failure, but they establish the baseline behavior to preserve
  - _Requirements: 3.1, 3.2, 3.3_

- [x] 3. Fix webhook signature verification by preventing body consumption

  - [x] 3.1 Implement the fix in routes.go
    - Modify `RegisterWebhookRoute` in `cmd/api/internal/billing/stripe/routes.go`
    - Read the raw request body bytes directly from `c.Request.Body` before calling `wh.Handle`
    - Store the body bytes in a variable
    - Restore the body to `c.Request.Body` using `io.NopCloser(bytes.NewBuffer(body))`
    - Handle read errors by returning HTTP 400 "bad request"
    - Pass the restored request to `wh.Handle(c.Writer, c.Request)`
    - Preserve the existing `http.MaxBytesReader` protection in the webhook handler
    - _Bug_Condition: isBugCondition(input) where input.path == "/api/v1/webhooks/stripe" AND input.method == "POST" AND input.header["Stripe-Signature"] EXISTS AND requestBodyConsumedByMiddleware(input)_
    - _Expected_Behavior: Signature verification succeeds using complete raw request body bytes, webhook processes event and returns HTTP 200_
    - _Preservation: Webhook event processing logic (subscription.created, subscription.updated, subscription.deleted, invoice.payment_succeeded, invoice.payment_failed) remains unchanged; invalid signatures still rejected with HTTP 400; other API endpoints unaffected_
    - _Requirements: 2.1, 2.2, 2.3, 3.1, 3.2, 3.3_

  - [x] 3.2 Verify bug condition exploration test now passes
    - **Property 1: Expected Behavior** - Webhook Signature Verification Succeeds
    - **IMPORTANT**: Re-run the SAME test from task 1 - do NOT write a new test
    - The test from task 1 encodes the expected behavior
    - When this test passes, it confirms the expected behavior is satisfied
    - Run bug condition exploration test from step 1
    - **EXPECTED OUTCOME**: Test PASSES (confirms bug is fixed - signature verification now succeeds for valid webhook events)
    - _Requirements: 2.1, 2.2, 2.3_

  - [x] 3.3 Verify preservation tests still pass
    - **Property 2: Preservation** - Webhook Event Processing Unchanged
    - **IMPORTANT**: Re-run the SAME tests from task 2 - do NOT write new tests
    - Run preservation property tests from step 2
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions - webhook event processing logic preserved, invalid signatures still rejected)
    - Confirm all tests still pass after fix (no regressions)
    - _Requirements: 3.1, 3.2, 3.3_

- [x] 4. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.
