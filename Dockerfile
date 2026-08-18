FROM golang:alpine AS builder
WORKDIR /app

RUN apk add --no-cache git bash

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG BUILT_AT
ARG GIT_COMMIT
ARG VERSION

RUN builtAt=${BUILT_AT:-$(date +'%F %T %z')} && \
    git fetch origin >/dev/null 2>&1; \
    gitCommit=${GIT_COMMIT:-$(git rev-parse --short origin/HEAD 2>/dev/null || git log --pretty=format:"%h" -1)} && \
    version=${VERSION:-$(git describe --abbrev=0 --tags 2>/dev/null || echo "v26.0.0.1")} && \
    ldflags="\
    -w -s \
    -X 'github.com/wwwangzilin/LotsACG-Standalone/internal/common/version.BuildTime=$builtAt' \
    -X 'github.com/wwwangzilin/LotsACG-Standalone/internal/common/version.Commit=$gitCommit' \
    -X 'github.com/wwwangzilin/LotsACG-Standalone/internal/common/version.Version=$version'\
    " && \
    CGO_ENABLED=0 go build -tags nodynamic -ldflags "$ldflags" -o lotsacg

FROM alpine:latest
WORKDIR /opt/lotsacg/

RUN apk add --no-cache bash ca-certificates ffmpeg && update-ca-certificates

COPY --from=builder /app/lotsacg .

ENTRYPOINT ["./lotsacg"]
