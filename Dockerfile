FROM golang:1.24 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -o iommufd-device-plugin ./cmd/main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/iommufd-device-plugin .
USER 65532:65532

ENTRYPOINT ["/iommufd-device-plugin"]
CMD ["-log-level=info"]
