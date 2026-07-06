# Runs `go test -race` inside a Linux container (the Windows race detector
# needs a C toolchain; Docker sidesteps that). Requires Docker Desktop running.
#
# Usage:
#   .\test-race.ps1                                  # whole repo
#   .\test-race.ps1 ./01-goroutines/...              # one module
#   .\test-race.ps1 -run TestCheckAll ./01-goroutines/exercises   # one test
#   .\test-race.ps1 -count=5 ./01-goroutines/...     # repeat runs
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$TestArgs
)

if (-not $TestArgs) { $TestArgs = @('./...') }

docker run --rm `
    -v "${PSScriptRoot}:/app" `
    -w /app `
    -v go-concurrency-build-cache:/root/.cache/go-build `
    -v go-concurrency-mod-cache:/go/pkg/mod `
    golang:1.26 `
    go test -race @TestArgs

exit $LASTEXITCODE
