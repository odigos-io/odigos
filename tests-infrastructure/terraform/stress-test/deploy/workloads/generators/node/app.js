import { trace } from '@opentelemetry/api';

const tracer = trace.getTracer('node-span-gen');

// Configurable knobs
const spansPerSec = Number(process.env.SPANS_PER_SEC || '1000');
const spanBytes = Number(process.env.SPAN_BYTES || '1000');

// Create payload
const attrPayload = 'x'.repeat(spanBytes);

// Log startup information
console.log(`[node-span-gen] Starting with ${spansPerSec} spans/sec, ${spanBytes} bytes per span`);
console.log(`[node-span-gen] OTEL_SERVICE_NAME: ${process.env.OTEL_SERVICE_NAME || 'Not set'}`);
console.log(`[node-span-gen] OTEL_RESOURCE_ATTRIBUTES: ${process.env.OTEL_RESOURCE_ATTRIBUTES || 'Not set'}`);

let totalSpans = 0;

function emitBatch(n) {
  for (let i = 0; i < n; i++) {
    const span = tracer.startSpan('node-span');
    span.setAttribute('payload', attrPayload);
    span.end();
  }
  
  totalSpans += n;
  if (totalSpans % 1000 === 0) {
    console.log(`[node-span-gen] Completed batch: Generated ${totalSpans} spans`);
  }
}

// Generate spans every second
setInterval(() => emitBatch(spansPerSec), 1000);

// Keep alive
setInterval(() => {}, 1 << 30);
