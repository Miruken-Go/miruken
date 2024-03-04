# Miruken
Core Miruken

## Docker

### Manual 
    docker pull golang:1.22
    docker run -it -v $(pwd):/go/src --workdir=/go/src golang:1.22
    CGO_ENABLED=0 go build
    go test ./... -count=1

### Automated
    docker run -v $(pwd):/go/src --workdir=/go/src golang:1.22 go test ./...
