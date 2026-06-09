// k6 BREAKPOINT test for workflow <WORKFLOW_PLACEHOLDER>.
//
// BREAKPOINT: ramping arrival rate, find the ceiling. PROTOCOL-ONLY -- a single
// browser VU mid-test adds noise to the signal we're hunting. Document the
// abort point (final iteration rate or arrival rate) as the breakpoint.
//
// `ramping-arrival-rate` (not VUs) because the cliff is in requests-per-second,
// not concurrent users. VU-based ramps are bounded by iteration duration; if
// iterations slow down, VUs back off and you never reach the cliff. Arrival
// rate injects requests on a fixed schedule.
//
// `abortOnFail` thresholds stop the run when SLOs break. `delayAbortEval` 30s
// avoids tripping on instantaneous fluctuations.
//
// COST WARNING (Grafana Cloud k6): breakpoint runs can use 30-100+ VUh.
// Confirm with the customer per runbook.md before pushing.
//
// Iteration body is identical to smoke.js's protocolIteration (no browser body
// here).
//
// Run:
//   ./tools/run-with-monitor.sh                tests/<WORKFLOW_PLACEHOLDER>/breakpoint.js
//   k6 cloud run                               tests/<WORKFLOW_PLACEHOLDER>/breakpoint.js   # confirm budget

import http from 'k6/http';
import { check, group, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'https://target-host.example';
const USER_AGENT =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 ' +
  '(KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36';

export const options = {
  userAgent: USER_AGENT,
  scenarios: {
    protocol: {
      executor: 'ramping-arrival-rate',
      exec: 'protocolIteration',
      startRate: 5,
      timeUnit: '1s',
      preAllocatedVUs: 50,
      maxVUs: 500,
      stages: [
        { duration: '20m', target: 500 },    // ramp arrival rate 5/s → 500/s
      ],
    },
    // NO browser scenario -- breakpoint hunts protocol throughput. Adding a
    // browser VU adds noise without contributing signal.
  },
  thresholds: {
    // abortOnFail: stop the run when SLOs break -- that's the breakpoint.
    http_req_failed: [
      { threshold: 'rate<0.05', abortOnFail: true, delayAbortEval: '30s' },
    ],
    http_req_duration: [
      { threshold: 'p(95)<2000', abortOnFail: true, delayAbortEval: '30s' },
    ],
    checks: ['rate>0.95'],
  },
};

// --- Protocol iteration ---------------------------------------------------------
// Body identical to smoke.js's protocolIteration. See smoke.js for the full
// template and hybrid-load-design.md for why duplication is preferred over a
// shared helper.

export function protocolIteration() {
  group('<WORKFLOW_PLACEHOLDER>', function () {
    let res = http.get(`${BASE_URL}/`, {
      headers: {
        Accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
      },
    });
    check(res, { 'homepage 200': (r) => r.status === 200 });

    // ... fill in from protocol.js with check() + tags. See smoke.js template.

    sleep(Math.random() * 3 + 1);

    // The user action (tagged)
  });
}
