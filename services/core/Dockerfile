FROM golang:1.23-alpine AS build
WORKDIR /src
COPY services/core/go.mod services/core/go.sum ./
RUN go mod download
COPY services/core/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /core ./cmd/core

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /core /usr/local/bin/core
USER 10001:10001
EXPOSE 9090
ENTRYPOINT ["/usr/local/bin/core"]
