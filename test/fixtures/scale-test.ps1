param (
    [string]$Action = "up"
)

if ($Action -eq "up") {
    Write-Host "Spawning 50 test containers for Orbit scale verification..."
    1..50 | ForEach-Object {
        docker run -d --name "orbit-scale-test-$_" --label "orbit.test=scale" --label "com.docker.compose.project=scale-bench" alpine:latest sh -c "while true; do sleep 30; done" | Out-Null
    }
    Write-Host "50 test containers started."
} elseif ($Action -eq "down") {
    Write-Host "Cleaning up 50 test containers..."
    $ids = docker ps -q --filter "label=orbit.test=scale"
    if ($ids) {
        docker rm -f $ids | Out-Null
    }
    Write-Host "Cleanup complete."
}
