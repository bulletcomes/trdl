# Operations

Local setup is in the [README quickstart](../README.md#quickstart-k3d). Check any deployment with `scripts/smoke-test.sh <url> [expected-tag]`.

## Deploying to GKE

Currently deployed to the Autopilot cluster `trdl-poc` in project `trdl-515015`, at `https://136-82-211-146.sslip.io/`.

1. Create a regional Autopilot cluster and connect. kubectl needs `gke-gcloud-auth-plugin`.
   ```bash
   gcloud container clusters create-auto trdl-poc --region europe-north1
   gcloud container clusters get-credentials trdl-poc --region europe-north1
   ```
2. Reserve the static IP: `gcloud compute addresses create trdl-ip --global`.
3. In `deployment/overlays/gke`, set the hostname in `ingress-patch.yaml` and `managedcertificate.yaml`, `newTag` in `kustomization.yaml`, and `projectId` in `scaledobject.yaml`. Without a domain, `<ip-with-dashes>.sslip.io` resolves to the IP.
4. Install KEDA, and let it read Cloud Monitoring:
   ```bash
   kubectl apply --server-side -f https://github.com/kedacore/keda/releases/download/v2.21.0/keda-2.21.0.yaml
   gcloud projects add-iam-policy-binding PROJECT_ID --role roles/monitoring.viewer \
     --member "principal://iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/PROJECT_ID.svc.id.goog/subject/ns/keda/sa/keda-operator"
   ```
5. Deploy, and check one pod per zone:
   ```bash
   kubectl apply -k deployment/overlays/gke
   kubectl -n trdl get pods -o wide && kubectl get nodes -L topology.kubernetes.io/zone
   ```
6. After 15–60 minutes (load balancer and certificate), run `scripts/smoke-test.sh https://<host> <tag>`. Test with the script rather than waiting for `kubectl get managedcertificate` to say `Active`; it lags.

Tear down: `gcloud container clusters delete trdl-poc --region europe-north1` and `gcloud compute addresses delete trdl-ip --global`.

## Autoscaling demo (k3d, optional)

```bash
kubectl apply --server-side -f https://github.com/kedacore/keda/releases/download/v2.21.0/keda-2.21.0.yaml
kubectl apply -k deployment/addons/prometheus
kubectl apply -k deployment/overlays/local-keda
kubectl apply -k deployment/loadtest           # k6: ramps to 400 req/s for 90 s
kubectl -n trdl get hpa,pods -w
```

Expected: 2 → 4 replicas within about a minute, back to 2 about 40 s after the load stops, and 0 failed requests in `kubectl -n trdl logs job/trdl-loadtest`. To rerun, delete the Job first. Stopping Prometheus (`kubectl -n monitoring scale deploy prometheus --replicas=0`) shows the fallback: 4 replicas.

## Rollouts and troubleshooting

```bash
kubectl -n trdl rollout status deployment/trdl
kubectl -n trdl get scaledobject,hpa,pods
kubectl -n trdl logs deployment/trdl
kubectl -n trdl rollout undo deployment/trdl   # with Flux in production: git revert instead
```

- **`404 page not found` on `localhost:8080`:** that's Traefik, not the service. The `gke` overlay's host rule was applied to k3d; fix it with `kubectl apply -k deployment/overlays/local`.
- **ScaledObject `READY=False`:** `kubectl -n trdl describe scaledobject trdl` shows why (metric source down, wrong query, missing permission). With `FALLBACK=True` it runs a fixed replica count until the metric is readable again.
