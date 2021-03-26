# Build the doctor binary
FROM golang:1.16 as builder

WORKDIR /github.com/wandera/spot-termination-handler

# this will cache the go mod download step, unless go.mod or go.sum changes
ENV GO111MODULE=on

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

# Build
RUN CGO_ENABLED=0 go build -v -a -o spot-termination-handler

FROM scratch
COPY --from=builder /github.com/wandera/spot-termination-handler/spot-termination-handler /bin/spot-termination-handler

ENTRYPOINT ["spot-termination-handler"]
