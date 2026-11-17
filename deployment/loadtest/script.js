// Ramps up to 400 requests/second against GET /, holds, then stops.
// Goes through Traefik so the requests show up in the metric KEDA scales on.
import http from "k6/http";
import { check } from "k6";

export const options = {
  scenarios: {
    ramp: {
      executor: "ramping-arrival-rate",
      startRate: 10,
      timeUnit: "1s",
      preAllocatedVUs: 50,
      maxVUs: 200,
      stages: [
        { target: 400, duration: "30s" },
        { target: 400, duration: "90s" },
        { target: 0, duration: "10s" },
      ],
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
  },
};

export default function () {
  const res = http.get("http://traefik.kube-system.svc/");
  check(res, { "status 200": (r) => r.status === 200, "body 42": (r) => r.body === "42" });
}
