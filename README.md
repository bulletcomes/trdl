# The Responding Dark Laughter (TRDL)

A small, production-minded proof of concept for an HTTP service that returns `42`.

This repository is designed to be runnable on a local Kubernetes cluster (k3d) while keeping the manifests and documentation understandable as a path toward a managed public-cloud Kubernetes platform such as Google Kubernetes Engine (GKE).

## What is included

- A minimal Go HTTP service.
- Unit tests for the HTTP contract and error cases.
- A non-root, multi-stage container image.
- Kubernetes Deployment and Service manifests managed with Kustomize.
- A base deployed directly to k3d, and a GKE-oriented overlay.
- A Traefik Ingress (k3d's built-in controller); the GKE overlay adds a host rule.
- GitHub Actions CI that tests, validates manifests, builds, scans, and publishes an image to GHCR.
- Operational, architecture, security, and production-readiness notes.

## Service contract

| Request | Expected result |
|---|---|
| `GET /` | `200`, body `42`, content type `text/plain; charset=utf-8` |
| `GET /healthz` | `200`, body `ok` |
| `GET /readyz` | `200`, body `ready` |
| `GET /version` | `200`, body is the build version (`dev` unless set at build time) |
| Any other path | `404` |
| Any non-GET method | `405`, with `Allow: GET` |

The service is intentionally stateless and has no database or other runtime dependency.

## Run locally without Kubernetes

Requirements: Go 1.22+.

```bash
cd app
go test ./...
go run .
```

In another terminal:

```bash
curl -i http://localhost:8080/
curl -i http://localhost:8080/version
```

## Build and run the container

```bash
docker build --build-arg VERSION=local -t trdl:local app
docker run --rm -p 8080:8080 trdl:local
curl -i http://localhost:8080/
curl -i http://localhost:8080/version
```

## Run on local Kubernetes (k3d)

Requirements: Docker, kubectl, k3d, and Kustomize (or a kubectl version with `kubectl apply -k`).

Create a local cluster:

```bash
k3d cluster create trdl \
  --servers 1 \
  --agents 2 \
  --port "8080:80@loadbalancer"
```

Deploy the application:

```bash
kubectl apply -k deployment/base
kubectl -n trdl rollout status deployment/trdl
kubectl -n trdl get pods -o wide
```

The base includes an Ingress for k3d's built-in Traefik controller, which the cluster's `8080:80@loadbalancer` port mapping exposes on the host. Access it at:

```bash
curl -i http://localhost:8080/
curl -i http://localhost:8080/version
```

Clean up:

```bash
kubectl delete namespace trdl
k3d cluster delete trdl
```

## CI/CD

The GitHub Actions workflow runs on pull requests and pushes to `main`:

1. Run Go unit tests.
2. Validate Kubernetes manifests with Kustomize.
3. Build the container.
4. Run a Trivy vulnerability scan.
5. On pushes to `main`, publish an immutable image tag (the commit SHA) to GitHub Container Registry (GHCR).

The workflow does **not** deploy to a cluster. A GitHub-hosted runner cannot directly reach a developer's laptop cluster, and this assignment environment does not assume cloud credentials. The GKE deployment path is documented separately rather than pretending a deployment has been performed.

To publish successfully, enable GitHub Actions package write permissions for the repository. The workflow uses the repository's built-in `GITHUB_TOKEN`.

## GKE direction

The `gke` overlay is a starting point, not a complete cloud provisioning solution. A production implementation would additionally provision/configure:

- A regional GKE cluster with nodes in multiple zones.
- Workload Identity Federation for CI/CD authentication.
- Artifact Registry (or GHCR) image access.
- A managed external HTTPS load balancer, DNS, and TLS certificate.
- Network policies, cloud logging/monitoring integration, and appropriate IAM.
- Topology spread constraints and disruption controls, validated against actual node labels and capacity.

The included Ingress targets Traefik, which k3d ships by default. On GKE, choose and configure the intended ingress controller/load-balancer implementation; do not assume this manifest alone provisions a production Google Cloud Load Balancer.

## Design choices and limitations

See [`docs/solution.md`](docs/solution.md), [`docs/architecture.md`](docs/architecture.md), and [`docs/operations.md`](docs/operations.md).

Important distinction: two replicas on a local three-node k3d cluster demonstrate Kubernetes scheduling and recovery behaviour, but do not prove regional high availability. The local cluster has a single control-plane server and does not model independent cloud zones, managed control-plane availability, external DNS, or provider failure domains.
