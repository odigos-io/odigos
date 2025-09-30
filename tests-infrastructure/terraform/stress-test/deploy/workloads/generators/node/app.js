import { trace } from '@opentelemetry/api';

const tracer = trace.getTracer('node-span-gen');

// Configurable knobs
const spansPerSec = Number(process.env.SPANS_PER_SEC || '6000');
const spanBytes = Number(process.env.SPAN_BYTES || '4000');

// Create payload
const attrPayload = 'x'.repeat(spanBytes);

// Log startup information
console.log(`[node-span-gen] Starting with ${spansPerSec} spans/sec, ${spanBytes} bytes per span`);
console.log(`[node-span-gen] OTEL_SERVICE_NAME: ${process.env.OTEL_SERVICE_NAME || 'Not set'}`);
console.log(`[node-span-gen] OTEL_RESOURCE_ATTRIBUTES: ${process.env.OTEL_RESOURCE_ATTRIBUTES || 'Not set'}`);

let totalSpans = 0;

function emitBatch(n) {
  for (let i = 0; i < n; i++) {
    const span = tracer.startSpan('load-span');
    span.setAttribute('payload', attrPayload);
    span.setAttribute('lang', 'node');
    span.setAttribute('gen', 'node-span-gen');
    span.setAttribute('payload_size', spanBytes);
    span.end();
    
    // Add small delay to reduce CPU usage (optional)
    if (i % 100 === 0) {
      Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, 1);
    }
  }
  
  totalSpans += n;
  console.log(`[node-span-gen] Generated ${n} spans in this second (total: ${totalSpans})`);
}

// Generate spans every second
setInterval(() => emitBatch(spansPerSec), 1000);

// Keep alive
setInterval(() => {}, 1 << 30);
