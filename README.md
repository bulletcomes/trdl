# The Responding Dark Laughter (TRDL)

A minimal Go HTTP service that answers `42`, with the container, Kubernetes manifests, CI and autoscaling needed to run it as a highly available service.

| Document | Contents |
|---|---|
| [docs/discovery.md](docs/discovery.md) | Requirements, assumptions, tasks and time estimate |
| [docs/solution.md](docs/solution.md) | Architecture, design choices, availability, autoscaling, security, limitations |
| [docs/operations.md](docs/operations.md) | Runbook: KEDA demo, GKE deployment, troubleshooting, rollouts |

## Service contract

| Request | Result |
|---|---|
| `GET /` | `200`, body `42`, `text/plain; charset=utf-8` |
| `GET /healthz` | `200`, body `ok` (liveness) |
| `GET /readyz` | `200`, body `ready` (readiness) |
| `GET /version` | `200`, the short commit SHA for CI-built images, `dev-0` otherwise |
| Any other path | `404` |
| Any non-GET method | `405`, with `Allow: GET` |

## Repository layout

```
app/                             Go service, tests, Dockerfile
deployment/base/                 Namespace, Deployment, Service, Traefik Ingress
deployment/overlays/local/       k3d: pinned image tag, spread across nodes
deployment/overlays/local-keda/  local + KEDA request-rate autoscaling
deployment/overlays/gke/         Reference: regional GKE over three zones (not deployed)
deployment/addons/prometheus/    Minimal Prometheus for the KEDA demo
deployment/loadtest/             k6 load test Job
```

## Quickstart (k3d)

Requires Docker, kubectl and k3d.

```bash
k3d cluster create trdl --servers 1 --agents 2 --port "8080:80@loadbalancer"
kubectl apply -k deployment/overlays/local
kubectl -n trdl rollout status deployment/trdl

curl -i http://localhost:8080/
curl -i http://localhost:8080/version
```

Use the `local` overlay, not `base` (which pins an old image tag) or `gke` (GKE-only resources, and a host rule that breaks `localhost`). For the autoscaling demo, see [operations.md](docs/operations.md#autoscaling-demo-with-keda-k3d).

Clean up with `k3d cluster delete trdl`.

## Development

Requires Go 1.27.

```bash
cd app && go test -race ./...               # unit tests
docker build -t trdl:local app              # container image
kubectl kustomize deployment/overlays/local # render manifests
```

Keep the service stateless, log to stdout, and add tests for new routes and error cases.

## CI/CD

GitHub Actions runs on pull requests and pushes to `main`: unit tests, a Kustomize render of every overlay, the image build, and a Trivy scan that fails on fixable HIGH/CRITICAL vulnerabilities. On `main` it publishes `docker.io/bulletcomes/trdl:<short-sha>`.

Publishing needs two repository secrets: `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` (a Docker Hub access token with Read & Write scope). The workflow does not deploy; see [solution.md](docs/solution.md#delivery).
