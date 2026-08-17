FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -o /out/seed ./cmd/seed

FROM alpine:3.20 AS seed
RUN apk add --no-cache ca-certificates
COPY --from=build /out/seed /usr/local/bin/seed
ENTRYPOINT ["/usr/local/bin/seed"]

# api is last so it's the default build target (`docker build` with no
# --target builds the final stage) — matters for platforms like Railway that
# build straight from this Dockerfile with no per-service target config.
FROM alpine:3.20 AS api
RUN apk add --no-cache ca-certificates
COPY --from=build /out/api /usr/local/bin/api
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
