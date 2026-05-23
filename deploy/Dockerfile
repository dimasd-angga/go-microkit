# Multi-stage build for any of the kit's services. Pass --build-arg SERVICE=<name>.
# Supported values: user, order, notification, gateway.
ARG GO_VERSION=1.22

FROM golang:${GO_VERSION}-alpine AS build
ARG SERVICE
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./$(if [ "$SERVICE" = "gateway" ]; then echo gateway; else echo services/$SERVICE; fi)

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
