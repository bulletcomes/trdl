# Developer guide

## Prerequisites

- Go 1.22+
- Docker
- For Kubernetes work: kubectl and k3d

## Typical change workflow

1. Create a branch.
2. Make the smallest change that satisfies the requirement.
3. Run `cd app && go test ./...`.
4. Build the container with `docker build -t trdl:local app`.
5. Render manifests with `kubectl kustomize deployment/base`.
6. Open a pull request and inspect the GitHub Actions results.

## Operational conventions

- Keep the application stateless.
- Log to stdout/stderr.
- Add tests for new routes and error cases.
- Keep image tags immutable in deployed environments.
- Update this guide when developer commands or required tooling change.
