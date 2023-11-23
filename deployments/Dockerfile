FROM --platform=$BUILDPLATFORM docker.io/golang:1.21 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

WORKDIR /app/cmd/cip-functions

ARG TARGETARCH
ARG TARGETOS
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build

FROM registry.access.redhat.com/ubi9/ubi-minimal
WORKDIR /opt/diwise

COPY --from=builder --chown=1001 /app/cmd/cip-functions/cip-functions /opt/diwise
COPY --chown=1001 assets/config/functions.csv /opt/diwise/config/functions.csv

RUN chown 1001 /opt/diwise

EXPOSE 8080
USER 1001

ENTRYPOINT ["/opt/diwise/cip-functions"]
