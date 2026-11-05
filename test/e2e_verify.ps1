$ErrorActionPreference = "Stop"
$base = "http://127.0.0.1:8787"
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

Write-Host "=== Orbit End-to-End Verification ===" -ForegroundColor Cyan

# 1. Health
$h = Invoke-RestMethod -Uri "$base/api/v1/health" -WebSession $session
Write-Host "[1/9] Healthcheck:" ($h | ConvertTo-Json -Compress)

# 2. Auth status
$s = Invoke-RestMethod -Uri "$base/api/v1/auth/status" -WebSession $session
Write-Host "[2/9] Auth status (initial):" ($s | ConvertTo-Json -Compress)

if ($s.needsSetup) {
    # Extract setup token from docker logs
    $logs = docker logs orbit-live 2>&1 | Out-String
    $match = [regex]::Match($logs, "SETUP TOKEN: ([a-f0-9]+)")
    $token = $match.Groups[1].Value
    Write-Host "[2.1] Setup token from logs:" $token

    # 3. Setup Admin
    $setupBody = @{ token = $token; username = "admin"; password = "OrbitAdminSecret2026!" } | ConvertTo-Json
    $setupResp = Invoke-RestMethod -Uri "$base/api/v1/auth/setup" -Method Post -Body $setupBody -ContentType "application/json" -WebSession $session
    Write-Host "[3/9] Setup Admin response:" ($setupResp | ConvertTo-Json -Compress)
} else {
    Write-Host "[2/9] Setup already completed previously."
}

# 4. Login
$loginBody = @{ username = "admin"; password = "OrbitAdminSecret2026!" } | ConvertTo-Json
$loginResp = Invoke-RestMethod -Uri "$base/api/v1/auth/login" -Method Post -Body $loginBody -ContentType "application/json" -WebSession $session
$csrf = $loginResp.csrfToken
Write-Host "[4/9] Login successful. CSRF Token received:" $csrf

# 5. List containers
$headers = @{ "X-CSRF-Token" = $csrf }
$containers = Invoke-RestMethod -Uri "$base/api/v1/containers" -Headers $headers -WebSession $session
Write-Host "[5/9] Discovered containers count:" $containers.Length
foreach ($c in $containers) {
    Write-Host ("  - {0,-22} State: {1,-8} Health: {2,-10} Compose: {3}" -f $c.name, $c.state, $c.healthStatus, $c.composeProject)
}

# 6. Container Metrics (wait briefly for stats collector)
Write-Host "Waiting 4s for stats collector tick..."
Start-Sleep -Seconds 4
$cpuContainer = $containers | Where-Object { $_.name -like "*cpu-load*" } | Select-Object -First 1
if ($cpuContainer) {
    $cpuStats = Invoke-RestMethod -Uri "$base/api/v1/containers/$($cpuContainer.id)/stats" -Headers $headers -WebSession $session
    Write-Host "[6/9] CPU-Load Container stats:" ("CPU: {0:N2}% | RAM: {1} MB" -f $cpuStats.current.cpuPercent, [Math]::Round($cpuStats.current.memoryBytes / 1MB, 2))
}

$memContainer = $containers | Where-Object { $_.name -like "*mem-load*" } | Select-Object -First 1
if ($memContainer) {
    $memStats = Invoke-RestMethod -Uri "$base/api/v1/containers/$($memContainer.id)/stats" -Headers $headers -WebSession $session
    Write-Host "[6/9] Mem-Load Container stats:" ("CPU: {0:N2}% | RAM: {1} MB | Limit: {2} MB" -f $memStats.current.cpuPercent, [Math]::Round($memStats.current.memoryBytes / 1MB, 2), [Math]::Round($memStats.current.memoryLimit / 1MB, 2))
}

# 7. Network Flow Topology
$netflow = Invoke-RestMethod -Uri "$base/api/v1/netflow" -Headers $headers -WebSession $session
Write-Host "[7/9] Netflow Topology: Nodes =" $netflow.nodes.Length "Edges =" $netflow.edges.Length "Sockets =" $netflow.activeSockets.Length

# 8. Container Actions (Stop / Start)
$healthyContainer = $containers | Where-Object { $_.name -like "*healthy*" } | Select-Object -First 1
if ($healthyContainer) {
    Write-Host "[8/9] Testing container STOP on:" $healthyContainer.name
    $stopResp = Invoke-RestMethod -Uri "$base/api/v1/containers/$($healthyContainer.id)/stop" -Method Post -Headers $headers -WebSession $session
    Write-Host "  Stop response:" ($stopResp | ConvertTo-Json -Compress)

    # Check updated container state
    Start-Sleep -Seconds 2
    $cCheck = Invoke-RestMethod -Uri "$base/api/v1/containers/$($healthyContainer.id)" -Headers $headers -WebSession $session
    Write-Host "  Container state after stop:" $cCheck.container.state

    Write-Host "  Testing container START on:" $healthyContainer.name
    $startResp = Invoke-RestMethod -Uri "$base/api/v1/containers/$($healthyContainer.id)/start" -Method Post -Headers $headers -WebSession $session
    Write-Host "  Start response:" ($startResp | ConvertTo-Json -Compress)

    Start-Sleep -Seconds 2
    $cCheck2 = Invoke-RestMethod -Uri "$base/api/v1/containers/$($healthyContainer.id)" -Headers $headers -WebSession $session
    Write-Host "  Container state after start:" $cCheck2.container.state
}

# 9. Audit Log
$audit = Invoke-RestMethod -Uri "$base/api/v1/audit" -Headers $headers -WebSession $session
Write-Host "[9/9] Audit Log Entries count:" $audit.Length
foreach ($entry in ($audit | Select-Object -First 5)) {
    Write-Host ("  - [{0}] Action: {1} | Target: {2} | Result: {3}" -f $entry.timestamp, $entry.action, $entry.target, $entry.result)
}

Write-Host "=== End-to-End Verification PASSED ===" -ForegroundColor Green

