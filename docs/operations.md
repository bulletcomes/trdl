# Operations

Local setup is in the [README quickstart](../README.md#quickstart-k3d).

## Autoscaling demo with KEDA (k3d)

KEDA scales on Traefik's request rate, read from a small Prometheus: one replica per 100 req/s, 2 to 6 replicas. Install once per cluster:

```bash
kubectl apply --server-side -f https://github.com/kedacore/keda/releases/download/v2.21.0/keda-2.21.0.yaml
kubectl apply -k deployment/addons/prometheus
kubectl -n keda rollout status deployment/keda-operator
kubectl -n monitoring rollout status deployment/prometheus
```

Deploy with autoscaling, run the load test (k6, ramping to 400 req/s for 90 s), and watch:

```bash
kubectl apply -k deployment/overlays/local-keda
kubectl apply -k deployment/loadtest
kubectl -n trdl get hpa,pods -w
kubectl -n trdl logs -f job/trdl-loadtest    # k6 summary at the end
```

The Job deletes itself 10 minutes after it finishes. To rerun sooner: `kubectl -n trdl delete job trdl-loadtest`.

Run on 2026-11-17, on 3-node k3d:

| Time | Load | Ready replicas |
|---|---|---:|
| 0:00 | ramp starts | 2 |
| 0:40 | ~115 req/s per pod | 3 |
| 1:00 | ~400 req/s | 4 |
| 2:15 | load stops | 4 |
| 2:37 | none | 3 |
| 2:52 | none | 2 |

k6 sent 44,150 requests, **0 failed**, p95 latency 0.41 ms. The scale-down window is shortened to 30 s for the demo; the GKE overlay keeps the 300 s default.

To see the fallback, run `kubectl -n monitoring scale deploy prometheus --replicas=0`. Within a minute the ScaledObject shows `FALLBACK=True` and the Deployment runs 4 replicas. Scale Prometheus back to 1 to return to normal scaling.

Remove the demo with `kubectl delete -k deployment/addons/prometheus` and `kubectl delete -f <the KEDA URL above>`, or delete the cluster.

## Autoscaling operations

KEDA owns the replica count through the HPA `keda-hpa-trdl`. Don't add another HPA or scale by hand; KEDA overwrites it.

```bash
kubectl -n trdl get scaledobject trdl          # READY, ACTIVE, FALLBACK
kubectl -n trdl get hpa keda-hpa-trdl          # metric value vs target
kubectl -n trdl describe scaledobject trdl     # conditions and events
kubectl -n keda logs deploy/keda-operator      # scaler errors
```

| Symptom | Likely cause | Check |
|---|---|---|
| `READY=False`, HPA target `<unknown>` | Metric unreadable: Prometheus or Cloud Monitoring down, bad query, missing permission | `KEDAScalerFailed` events, operator logs |
| `FALLBACK=True`, 4 (local) or 6 (GKE) replicas | 3 failed metric reads in a row | As above; ends by itself when the metric is readable again |
| Stuck at minimum under load | Query matches nothing (for example, the Traefik service label changed) | Run the query in Prometheus: `sum(rate(traefik_service_requests_total{service="trdl-trdl-80@kubernetes"}[30s]))` |
| Replicas flapping | Scale-down window too short | `advanced.horizontalPodAutoscalerConfig.behavior.scaleDown` |

To pin the replica count during an incident, and to resume:

```bash
kubectl -n trdl annotate scaledobject trdl autoscaling.keda.sh/paused-replicas="4" --overwrite
kubectl -n trdl annotate scaledobject trdl autoscaling.keda.sh/paused-replicas-
```

## Deploying to GKE

`deployment/overlays/gke` is a reference; it has not been applied to a real cluster. The design is in [solution.md](solution.md#availability).

1. Create a regional cluster, so nodes are in several zones:
   ```bash
   gcloud container clusters create-auto trdl --region europe-north1
   ```
2. Reserve the static IP the Ingress uses:
   ```bash
   gcloud compute addresses create trdl-ip --global
   ```
3. Replace `trdl.example.com` in `ingress-patch.yaml` and `managedcertificate.yaml`, and point the domain's DNS A record at the static IP.
4. Set `newTag` in `kustomization.yaml` and `projectId` in `scaledobject.yaml`.
5. Install KEDA and let it read Cloud Monitoring through Workload Identity:
   ```bash
   kubectl apply --server-side -f https://github.com/kedacore/keda/releases/download/v2.21.0/keda-2.21.0.yaml
   gcloud projects add-iam-policy-binding PROJECT_ID \
     --role roles/monitoring.viewer \
     --member "principal://iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/PROJECT_ID.svc.id.goog/subject/ns/keda/sa/keda-operator"
   kubectl -n keda rollout restart deployment/keda-operator
   ```
6. Deploy, and check that the pods are in different zones:
   ```bash
   kubectl apply -k deployment/overlays/gke
   kubectl -n trdl get pods -o wide
   kubectl get nodes -L topology.kubernetes.io/zone
   ```

The managed certificate becomes active only after DNS resolves to the load balancer, which can take 15 to 60 minutes.

## Troubleshooting

### `404 page not found` on `localhost:8080`

That body comes from Traefik, not the service: the Ingress has a host rule that `localhost` doesn't match. `kubectl -n trdl get ingress trdl` shows a host instead of `*`. This usually means the `gke` overlay was applied. Fix it with `kubectl apply -k deployment/overlays/local`, or test with `curl -H 'Host: trdl.example.com' http://localhost:8080/`.

### `FailedPreStopHook` events

These came from the old `preStop` hook running `/bin/sh` in a distroless image. The base now uses the built-in `sleep` action (Kubernetes 1.30+). If you still see them, re-apply the overlay.

## Rollouts and recovery

```bash
kubectl -n trdl rollout status deployment/trdl
kubectl -n trdl get deploy,pods,endpointslices
kubectl -n trdl logs deployment/trdl
kubectl -n trdl rollout history deployment/trdl
kubectl -n trdl rollout undo deployment/trdl
```

Image tags are immutable short SHAs, so a rollback is a redeploy of the previous tag.

## SLO discussion

Agree a target only after the requirements and measurement are defined. Annual downtime budgets, for reference:

| Availability | Downtime per year |
|---|---:|
| 99% | 3 days 15 hours |
| 99.9% | 8 hours 46 minutes |
| 99.99% | 52 minutes 34 seconds |

Define the measurement window, exclusions and responsibility boundaries before committing to any of these.
