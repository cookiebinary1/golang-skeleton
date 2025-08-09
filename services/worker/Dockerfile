# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS base
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

FROM base AS dev
RUN --mount=type=cache,target=/go/pkg/mod go install github.com/air-verse/air@v1.61.1
CMD ["air", "-c", "services/worker/.air.toml"]

FROM base AS build
COPY pkg ./pkg
COPY services/worker ./services/worker
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./services/worker/cmd

FROM gcr.io/distroless/static-debian12:nonroot AS release
COPY --from=build /out/worker /worker
USER nonroot:nonroot
ENTRYPOINT ["/worker"]
