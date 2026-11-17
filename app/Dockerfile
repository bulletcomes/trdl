# Build a small static binary.
FROM golang:1.27.4-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/trdl .

# Distroless runtime: no shell or package manager in the final image.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/trdl /trdl
EXPOSE 8080
# Numeric UID/GID (distroless "nonroot") so Kubernetes runAsNonRoot can verify it.
USER 65532:65532
ENTRYPOINT ["/trdl"]
