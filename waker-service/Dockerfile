# Edge, for Railway (service root dir: waker-service/). Mount a volume at
# /data so tsnet's node state survives redeploys.
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /edge-api ./cmd/edge-api

# distroless/static ships the CA certs needed for HTTPS to *.ts.net.
FROM gcr.io/distroless/static
COPY --from=build /edge-api /edge-api
ENV TS_STATE_DIR=/data/tsnet
ENTRYPOINT ["/edge-api"]
