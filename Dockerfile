ARG BASE_IMAGE=gcr.io/distroless/static-debian12:nonroot

FROM golang:1.24 AS builder

ARG GIT_SHA=unknown
ARG GIT_DIRTY=""

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux make build

FROM ${BASE_IMAGE}

WORKDIR /app

COPY --from=builder /build/bin/hyperfleet-cloud-provider /app/hyperfleet-cloud-provider

COPY --from=builder /build/examples/kubeconfig /app/examples/kubeconfig

ENTRYPOINT ["/app/hyperfleet-cloud-provider"]
CMD ["--help"]

LABEL name="hyperfleet-cloud-provider" \
      vendor="Red Hat" \
      version="0.0.1" \
      summary="HyperFleet Cloud Provider - Multi-cloud Kubernetes Token Provider" \
      description="Kubernetes authentication token provider for GKE, EKS, and AKS without cloud CLIs"
