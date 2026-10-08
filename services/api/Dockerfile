FROM golang:1.23-alpine AS build
WORKDIR /src
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /api /usr/local/bin/api
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
