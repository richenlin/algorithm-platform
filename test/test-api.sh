#!/bin/bash

# API Endpoint Test Script
# Algorithm Platform API Testing

BASE_URL="http://localhost:8080"
ALGORITHMS_ENDPOINT="/api/v1/algorithms"
JOBS_ENDPOINT="/api/v1/jobs"
DATA_ENDPOINT="/api/v1/data"

echo "=========================================="
echo "Algorithm Platform API Test"
echo "=========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test helper
test_endpoint() {
    local name=$1
    local method=$2
    local endpoint=$3
    local data=$4
    
    echo -e "${YELLOW}Testing${NC}: $name"
    echo "Endpoint: $method $endpoint"
    
    if [ -z "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -H "Origin: http://localhost:5173")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -H "Origin: http://localhost:5173" \
            -d "$data")
    fi
    
    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
        echo -e "${GREEN}✓ Success${NC} (HTTP $http_code)"
        echo "Response: $body"
    else
        echo -e "${RED}✗ Failed${NC} (HTTP $http_code)"
        echo "Response: $body"
    fi
    echo ""
}

# Test 1: List Algorithms
test_endpoint "List Algorithms" "GET" "$ALGORITHMS_ENDPOINT"

# Test 2: Create Algorithm
test_endpoint "Create Algorithm" "POST" "$ALGORITHMS_ENDPOINT" '{
  "name": "Test Algorithm",
  "description": "This is a test algorithm for API validation",
  "language": "python",
  "platform": "docker",
  "category": "ml",
  "entrypoint": "main.py"
}'

# Test 3: List Algorithms (after creation)
test_endpoint "List Algorithms (with data)" "GET" "$ALGORITHMS_ENDPOINT"

# Test 4: Get Algorithm Details (assuming ID alg_001)
test_endpoint "Get Algorithm Details" "GET" "$ALGORITHMS_ENDPOINT/alg_001"

# Test 5: Update Algorithm
test_endpoint "Update Algorithm" "PUT" "$ALGORITHMS_ENDPOINT/alg_001" '{
  "name": "Updated Test Algorithm",
  "description": "Updated description",
  "category": "cv"
}'

# Test 6: Create Version
test_endpoint "Create Version" "POST" "$ALGORITHMS_ENDPOINT/alg_001/versions" '{
  "source_code_zip_url": "http://minio:9000/algorithms/test.zip",
  "commit_message": "Initial version"
}'

# Test 7: Rollback Version
test_endpoint "Rollback Version" "POST" "$ALGORITHMS_ENDPOINT/alg_001/versions/v1/rollback" '{}'

# Test 8: List Jobs
test_endpoint "List Jobs" "GET" "$JOBS_ENDPOINT"

# Test 9: Get Job Details
test_endpoint "Get Job Details" "GET" "$JOBS_ENDPOINT/job_001/detail"

# Test 10: List Data
test_endpoint "List Data" "GET" "$DATA_ENDPOINT"

# Test 11: Upload Data
test_endpoint "Upload Data" "POST" "$DATA_ENDPOINT/upload" '{
  "filename": "test_data.csv",
  "category": "input",
  "minio_path": "data/input/test_data.csv"
}'

# Test 12: Execute Algorithm
test_endpoint "Execute Algorithm" "POST" "$ALGORITHMS_ENDPOINT/alg_001/execute" '{
  "mode": "batch",
  "params": {"param1": "value1"},
  "input_source": {"type": "minio", "url": "http://minio:9000/data/input/test.csv"},
  "resource_config": {"cpu_limit": 2.0, "memory_limit": "4Gi"},
  "timeout_seconds": 300
}'

echo "=========================================="
echo "CORS Test"
echo "=========================================="
echo ""

# Test CORS
echo "Testing CORS Preflight (OPTIONS)..."
response=$(curl -s -w "\n%{http_code}" -X OPTIONS "$BASE_URL$ALGORITHMS_ENDPOINT" \
            -H "Origin: http://localhost:5173" \
            -H "Access-Control-Request-Method: GET" \
            -H "Access-Control-Request-Headers: Content-Type")

http_code=$(echo "$response" | tail -n 1)

if [ "$http_code" = "200" ]; then
    echo -e "${GREEN}✓ CORS Preflight Success${NC}"
else
    echo -e "${RED}✗ CORS Preflight Failed${NC}"
fi

echo ""
echo "=========================================="
echo "Summary"
echo "=========================================="
echo "All tests completed."
echo "Check the responses above for details."
