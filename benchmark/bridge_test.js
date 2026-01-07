import http from 'k6/http';
import { Counter, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';
import exec from 'k6/execution';
import sse from 'k6/x/sse';
import encoding from 'k6/encoding';

export let sse_message_received = new Counter('sse_message_received');
export let sse_message_sent = new Counter('sse_message_sent');
export let sse_errors = new Counter('sse_errors');
export let post_errors = new Counter('post_errors');
export let delivery_latency = new Trend('delivery_latency');
export let json_parse_errors = new Counter('json_parse_errors');
export let missing_timestamps = new Counter('missing_timestamps');


const BRIDGE_URL = __ENV.BRIDGE_URL || 'http://localhost:8081/bridge';

// 1 minutes ramp-up, 1 minutes steady, 1 minutes ramp-down
const RAMP_UP = __ENV.RAMP_UP || '10s';
const HOLD = __ENV.HOLD || '10s';
const RAMP_DOWN = __ENV.RAMP_DOWN || '10s';

const SSE_VUS = Number(__ENV.SSE_VUS || 100);
const SEND_RATE = Number(__ENV.SEND_RATE || 1000);

const PRE_ALLOCATED_VUS = SSE_VUS
const MAX_VUS = PRE_ALLOCATED_VUS

const LISTENER_WRITERS_RATIO = Number(__ENV.LISTENER_WRITERS_RATIO || 3);
const TOTAL_INSTANCES = Number(__ENV.TOTAL_INSTANCES || 1);

const ID_SPACE_SIZE = SSE_VUS * TOTAL_INSTANCES * LISTENER_WRITERS_RATIO;

// Generate valid hex client IDs that the bridge expects
function getSSEIDs(vuIndex) {
  const startIndex = (vuIndex) * LISTENER_WRITERS_RATIO;
  const ids = [];
  const ids_dec = [];
  for (let i = 0; i < LISTENER_WRITERS_RATIO; i++) {
    const id = startIndex + i;
    ids_dec.push(id);
    ids.push([id.toString(16).padStart(64, '0')]);
  }
  return ids;
}

// Generate valid hex client IDs that the bridge expects
// This generates a random client ID for the sender in the ID space limited by vuIndex
// to avoid sending message to non-existent listener
// vuIndex is the incrementing index
function getID(vuIndex) {
  let maxIndex = ID_SPACE_SIZE - LISTENER_WRITERS_RATIO - 1;
  maxIndex = vuIndex < maxIndex ? vuIndex : maxIndex;
    
  const targetIndex = Math.floor(Math.random() * maxIndex);
  return targetIndex.toString(16).padStart(64, '0');
}

export const options = {
    discardResponseBodies: true,
    systemTags: ['status', 'method', 'name', 'scenario'], // Exclude 'url' to prevent metrics explosion
    thresholds: {
        http_req_failed: ['rate<0.01'],
        delivery_latency: ['p(95)<2000'],
        sse_errors: ['count<10'], // SSE should be very stable
        json_parse_errors: ['count<5'], // Should rarely fail to parse
        missing_timestamps: ['count<100'], // Most messages should have timestamps
        sse_message_sent: ['count>5'],
        sse_message_received: ['count>5'],

    },
    scenarios: {
        sse: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: RAMP_UP, target: SSE_VUS },   // warm-up
                { duration: HOLD, target: SSE_VUS },      // steady
                { duration: RAMP_DOWN, target: 0 },       // cool-down
            ],
            gracefulRampDown: '30s',
            exec: 'sseWorker'
        },
        senders: {
            executor: 'ramping-arrival-rate',
            startRate: 0,
            timeUnit: '1s',
            preAllocatedVUs: PRE_ALLOCATED_VUS,
            maxVUs: MAX_VUS,
            stages: [
                { duration: RAMP_UP, target: SEND_RATE }, // warm-up
                { duration: HOLD, target: SEND_RATE },    // steady
                { duration: RAMP_DOWN, target: 0 },       // cool-down
            ],
            gracefulStop: '30s',
            exec: 'messageSender'
        },
    },
};

export function sseWorker() {
  const vuIndex = exec.scenario.iterationInTest;
  const ids = getSSEIDs(vuIndex);
  const url = `${BRIDGE_URL}/events?client_id=${ids.join(',')}`;
  
  // Keep reconnecting for the test duration
  for (;;) {
    try {
      sse.open(url, { 
        headers: { Accept: 'text/event-stream' },
        tags: { name: 'SSE /events' }
      }, (c) => {
        c.on('event', (ev) => {
          if (ev.data === 'heartbeat' || !ev.data || ev.data.trim() === '') {
            return; // Skip heartbeats and empty events
          }
          try {
            // Parse the SSE event data first
            const eventData = JSON.parse(ev.data);
            // Then decode the base64 message field
            const decoded = encoding.b64decode(eventData.message, 'std', 's');
            const m = JSON.parse(decoded);
            if (m.ts) {
              const latency = Date.now() - m.ts;
              delivery_latency.add(latency);
              sse_message_received.add(1);
            } else {
              missing_timestamps.add(1);
              console.log('Message missing timestamp:', decoded);
            }
          } catch(e) {
            json_parse_errors.add(1);
            console.log('JSON parse error:', e, 'data:', ev.data);
          }
        });
        c.on('error', (err) => {
          console.log('SSE error:', err);
          sse_errors.add(1);
        });
      });
    } catch (e) {
      console.log('SSE connection failed:', e);
      sse_errors.add(1);
    }
  }
}

export function messageSender() {
  // Use fixed client pairs to reduce URL variations
  const vuIndex = exec.scenario.iterationInTest;
  const to = getID(vuIndex);
  let from = getID(vuIndex);
  // Avoid sending message to the same client ID
  while (from === to) {
    from = getID()
  }
  
  const topic = Math.random() < 0.5 ? 'sendTransaction' : 'signData';
  const body = encoding.b64encode(JSON.stringify({ ts: Date.now(), data: `${from} ${to}` }));
  const url = `${BRIDGE_URL}/message?client_id=${from}&to=${to}&ttl=300&topic=${topic}`;
  
  const r = http.post(url, body, {
    headers: { 'Content-Type': 'text/plain' },
    timeout: '10s',
    tags: { name: 'POST /message' }, // Group all message requests
  });
  if (r.status !== 200) {
    post_errors.add(1);
  } else {
    sse_message_sent.add(1);
  }
}
