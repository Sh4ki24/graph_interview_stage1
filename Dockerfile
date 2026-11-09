# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/app .

# ---- run stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl
RUN mkdir -p /app
COPY --from=build /bin/app /app/app
COPY migrations /migrations
COPY config.docker.yaml /app/config.yaml
EXPOSE 8080 9090
USER nobody
ENTRYPOINT ["/app/app"]
