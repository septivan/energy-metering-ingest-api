#!/bin/bash

# Test script for energy-metering-ingest-api

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "======================================"
echo "Energy Metering Ingest API Test Suite"
echo "======================================"
echo ""
echo "Testing endpoint: $BASE_URL"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test 1: Health Check
echo "Test 1: Health Check"
echo "---------------------"
response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")
if [ "$response" -eq 200 ]; then
    echo -e "${GREEN}✓ PASSED${NC} - Health endpoint returned 200"
else
    echo -e "${RED}✗ FAILED${NC} - Health endpoint returned $response"
fi
echo ""

# Test 2: Valid Meter Reading
echo "Test 2: Valid Meter Reading"
echo "----------------------------"
response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/meter/readings" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token" \
  -d '{
    "PM": [
      {
        "date": "29/12/2025 10:30:00",
        "data": "[230.5]",
        "name": "Volts"
      }
    ]
  }')

if [ "$response" -eq 202 ]; then
    echo -e "${GREEN}✓ PASSED${NC} - Valid reading accepted (202)"
else
    echo -e "${RED}✗ FAILED${NC} - Expected 202, got $response"
fi
echo ""

# Test 3: Multiple Readings
echo "Test 3: Multiple Readings"
echo "-------------------------"
response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/meter/readings" \
  -H "Content-Type: application/json" \
  -d '{
    "PM": [
      {
        "date": "29/12/2025 10:30:00",
        "data": "[230.5]",
        "name": "Volts"
      },
      {
        "date": "29/12/2025 10:30:05",
        "data": "[5.2]",
        "name": "Amps"
      },
      {
        "date": "29/12/2025 10:30:10",
        "data": "[1150.0]",
        "name": "Watts"
      }
    ]
  }')

if [ "$response" -eq 202 ]; then
    echo -e "${GREEN}✓ PASSED${NC} - Multiple readings accepted (202)"
else
    echo -e "${RED}✗ FAILED${NC} - Expected 202, got $response"
fi
echo ""

# Test 4: Missing PM Field
echo "Test 4: Missing PM Field (should fail)"
echo "---------------------------------------"
response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/meter/readings" \
  -H "Content-Type: application/json" \
  -d '{}')

if [ "$response" -eq 400 ]; then
    echo -e "${GREEN}✓ PASSED${NC} - Missing PM rejected (400)"
else
    echo -e "${RED}✗ FAILED${NC} - Expected 400, got $response"
fi
echo ""

# Test 5: Empty PM Array
echo "Test 5: Empty PM Array (should fail)"
echo "-------------------------------------"
response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/meter/readings" \
  -H "Content-Type: application/json" \
  -d '{"PM": []}')

if [ "$response" -eq 400 ] || [ "$response" -eq 503 ]; then
    echo -e "${GREEN}✓ PASSED${NC} - Empty PM array rejected ($response)"
else
    echo -e "${RED}✗ FAILED${NC} - Expected 400 or 503, got $response"
fi
echo ""

# Test 6: Missing Required Fields
echo "Test 6: Missing Required Fields (should fail)"
echo "----------------------------------------------"
response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/meter/readings" \
  -H "Content-Type: application/json" \
  -d '{
    "PM": [
      {
        "date": "29/12/2025 10:30:00"
      }
    ]
  }')

if [ "$response" -eq 400 ]; then
    echo -e "${GREEN}✓ PASSED${NC} - Missing fields rejected (400)"
else
    echo -e "${RED}✗ FAILED${NC} - Expected 400, got $response"
fi
echo ""

# Test 7: Invalid JSON
echo "Test 7: Invalid JSON (should fail)"
echo "-----------------------------------"
response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/meter/readings" \
  -H "Content-Type: application/json" \
  -d '{invalid json}')

if [ "$response" -eq 400 ]; then
    echo -e "${GREEN}✓ PASSED${NC} - Invalid JSON rejected (400)"
else
    echo -e "${RED}✗ FAILED${NC} - Expected 400, got $response"
fi
echo ""

# Test 8: With Authorization Header
echo "Test 8: With Authorization Header"
echo "----------------------------------"
response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/meter/readings" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" \
  -d '{
    "PM": [
      {
        "date": "29/12/2025 10:30:00",
        "data": "[230.5]",
        "name": "Volts"
      }
    ]
  }')

if [ "$response" -eq 202 ]; then
    echo -e "${GREEN}✓ PASSED${NC} - Request with auth header accepted (202)"
else
    echo -e "${RED}✗ FAILED${NC} - Expected 202, got $response"
fi
echo ""

echo "======================================"
echo "Test suite completed"
echo "======================================"
