#!/bin/bash

# Script to test the deployed API

API_ENDPOINT=${1}

if [ -z "$API_ENDPOINT" ]; then
    echo "Usage: ./test-api.sh <API_ENDPOINT>"
    echo "Example: ./test-api.sh https://abc123.execute-api.us-east-1.amazonaws.com/dev/payments"
    exit 1
fi

echo "==================================="
echo "Testing Crypto Conversion API"
echo "Endpoint: $API_ENDPOINT"
echo "==================================="

# Generate a unique idempotency key
IDEMPOTENCY_KEY=$(uuidgen | tr '[:upper:]' '[:lower:]')

echo ""
echo "Test 1: Create a payment"
echo "Idempotency Key: $IDEMPOTENCY_KEY"
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_ENDPOINT" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d '{
    "amount": 100000,
    "currency": "EUR",
    "source_account": "user123",
    "destination_account": "merchant456"
  }')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "HTTP Status: $HTTP_CODE"
echo "Response Body:"
echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"

if [ "$HTTP_CODE" -eq 202 ]; then
    echo ""
    echo "✅ Test 1 PASSED: Payment accepted"
    PAYMENT_ID=$(echo "$BODY" | jq -r '.payment_id' 2>/dev/null)
    echo "Payment ID: $PAYMENT_ID"
else
    echo ""
    echo "❌ Test 1 FAILED: Expected 202, got $HTTP_CODE"
fi

echo ""
echo "==================================="
echo ""
echo "Test 2: Duplicate idempotency key"
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_ENDPOINT" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d '{
    "amount": 100000,
    "currency": "EUR",
    "source_account": "user123",
    "destination_account": "merchant456"
  }')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "HTTP Status: $HTTP_CODE"
echo "Response Body:"
echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"

if [ "$HTTP_CODE" -eq 409 ]; then
    echo ""
    echo "✅ Test 2 PASSED: Duplicate request rejected"
else
    echo ""
    echo "❌ Test 2 FAILED: Expected 409, got $HTTP_CODE"
fi

echo ""
echo "==================================="
echo ""
echo "Test 3: Missing idempotency key"
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_ENDPOINT" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 100000,
    "currency": "EUR",
    "source_account": "user123",
    "destination_account": "merchant456"
  }')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "HTTP Status: $HTTP_CODE"
echo "Response Body:"
echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"

if [ "$HTTP_CODE" -eq 400 ]; then
    echo ""
    echo "✅ Test 3 PASSED: Missing header rejected"
else
    echo ""
    echo "❌ Test 3 FAILED: Expected 400, got $HTTP_CODE"
fi

echo ""
echo "==================================="
echo ""
echo "Test 4: Invalid amount (negative)"
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_ENDPOINT" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen | tr '[:upper:]' '[:lower:]')" \
  -d '{
    "amount": -1000,
    "currency": "EUR",
    "source_account": "user123",
    "destination_account": "merchant456"
  }')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "HTTP Status: $HTTP_CODE"
echo "Response Body:"
echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"

if [ "$HTTP_CODE" -eq 400 ]; then
    echo ""
    echo "✅ Test 4 PASSED: Invalid amount rejected"
else
    echo ""
    echo "❌ Test 4 FAILED: Expected 400, got $HTTP_CODE"
fi

echo ""
echo "==================================="
echo "Testing completed"
echo "==================================="
