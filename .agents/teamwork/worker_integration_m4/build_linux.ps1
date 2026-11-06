$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
Set-Location backend
go build -ldflags="-s -w" -o orbit ./cmd/orbit
