ARG GOPROXY="https://goproxy.cn"

# Build the function-agent binary
FROM golang:1.18 as builder
WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# Download go module
RUN go env -w GOPROXY=${GOPROXY} && go mod download

# Copy the go source
COPY main.go main.go

# Build
RUN GOPROXY=${GOPROXY} CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o agent main.go

# Use distroless as minimal base image to package the agent binary
FROM openfunction/distroless-static:nonroot
WORKDIR /
COPY --from=builder /workspace/agent .
USER 65532:65532

ENTRYPOINT ["/agent"]