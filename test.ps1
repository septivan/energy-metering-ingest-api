# PowerShell test script for energy-metering-ingest-api

$BASE_URL = if ($env:BASE_URL) { $env:BASE_URL } else { "http://localhost:8080" }

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "Energy Metering Ingest API Test Suite" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Testing endpoint: $BASE_URL"
Write-Host ""

# Test 1: Health Check
Write-Host "Test 1: Health Check" -ForegroundColor Yellow
Write-Host "---------------------"
try {
    $response = Invoke-WebRequest -Uri "$BASE_URL/health" -Method Get -UseBasicParsing
    if ($response.StatusCode -eq 200) {
        Write-Host "✓ PASSED - Health endpoint returned 200" -ForegroundColor Green
    }
} catch {
    Write-Host "✗ FAILED - Health endpoint error: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# Test 2: Valid Meter Reading
Write-Host "Test 2: Valid Meter Reading" -ForegroundColor Yellow
Write-Host "----------------------------"
$body = @{
    PM = @(
        @{
            date = "29/12/2025 10:30:00"
            data = "[230.5]"
            name = "Volts"
        }
    )
} | ConvertTo-Json

try {
    $response = Invoke-WebRequest -Uri "$BASE_URL/api/v1/meter/readings" `
        -Method Post `
        -ContentType "application/json" `
        -Body $body `
        -Headers @{ "Authorization" = "Bearer test-token" } `
        -UseBasicParsing
    
    if ($response.StatusCode -eq 202) {
        Write-Host "✓ PASSED - Valid reading accepted (202)" -ForegroundColor Green
    }
} catch {
    Write-Host "✗ FAILED - Expected 202, got error: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# Test 3: Multiple Readings
Write-Host "Test 3: Multiple Readings" -ForegroundColor Yellow
Write-Host "-------------------------"
$body = @{
    PM = @(
        @{
            date = "29/12/2025 10:30:00"
            data = "[230.5]"
            name = "Volts"
        },
        @{
            date = "29/12/2025 10:30:05"
            data = "[5.2]"
            name = "Amps"
        },
        @{
            date = "29/12/2025 10:30:10"
            data = "[1150.0]"
            name = "Watts"
        }
    )
} | ConvertTo-Json -Depth 3

try {
    $response = Invoke-WebRequest -Uri "$BASE_URL/api/v1/meter/readings" `
        -Method Post `
        -ContentType "application/json" `
        -Body $body `
        -UseBasicParsing
    
    if ($response.StatusCode -eq 202) {
        Write-Host "✓ PASSED - Multiple readings accepted (202)" -ForegroundColor Green
    }
} catch {
    Write-Host "✗ FAILED - Expected 202, got error: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# Test 4: Missing PM Field
Write-Host "Test 4: Missing PM Field (should fail)" -ForegroundColor Yellow
Write-Host "---------------------------------------"
$body = @{} | ConvertTo-Json

try {
    $response = Invoke-WebRequest -Uri "$BASE_URL/api/v1/meter/readings" `
        -Method Post `
        -ContentType "application/json" `
        -Body $body `
        -UseBasicParsing
    Write-Host "✗ FAILED - Should have returned 400" -ForegroundColor Red
} catch {
    if ($_.Exception.Response.StatusCode.value__ -eq 400) {
        Write-Host "✓ PASSED - Missing PM rejected (400)" -ForegroundColor Green
    } else {
        Write-Host "✗ FAILED - Expected 400, got $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Red
    }
}
Write-Host ""

# Test 5: Empty PM Array
Write-Host "Test 5: Empty PM Array (should fail)" -ForegroundColor Yellow
Write-Host "-------------------------------------"
$body = @{ PM = @() } | ConvertTo-Json

try {
    $response = Invoke-WebRequest -Uri "$BASE_URL/api/v1/meter/readings" `
        -Method Post `
        -ContentType "application/json" `
        -Body $body `
        -UseBasicParsing
    Write-Host "✗ FAILED - Should have returned 400 or 503" -ForegroundColor Red
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    if ($statusCode -eq 400 -or $statusCode -eq 503) {
        Write-Host "✓ PASSED - Empty PM array rejected ($statusCode)" -ForegroundColor Green
    } else {
        Write-Host "✗ FAILED - Expected 400 or 503, got $statusCode" -ForegroundColor Red
    }
}
Write-Host ""

# Test 6: Invalid JSON
Write-Host "Test 6: Invalid JSON (should fail)" -ForegroundColor Yellow
Write-Host "-----------------------------------"
try {
    $response = Invoke-WebRequest -Uri "$BASE_URL/api/v1/meter/readings" `
        -Method Post `
        -ContentType "application/json" `
        -Body "{invalid json}" `
        -UseBasicParsing
    Write-Host "✗ FAILED - Should have returned 400" -ForegroundColor Red
} catch {
    if ($_.Exception.Response.StatusCode.value__ -eq 400) {
        Write-Host "✓ PASSED - Invalid JSON rejected (400)" -ForegroundColor Green
    } else {
        Write-Host "✗ FAILED - Expected 400, got $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Red
    }
}
Write-Host ""

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "Test suite completed" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
