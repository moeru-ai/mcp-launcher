FROM golang:1.24 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o ./result/mcp-launcher ./cmd/mcp-launcher

# https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static-debian12 AS app

COPY --from=builder /app/result/mcp-launcher /

ENTRYPOINT ["/mcp-launcher"]
