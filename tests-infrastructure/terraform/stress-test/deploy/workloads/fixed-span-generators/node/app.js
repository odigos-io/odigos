import { trace } from '@opentelemetry/api';

const tracer = trace.getTracer('node-fixed-span-gen');

// Configurable knobs
const totalSpans = Number(process.env.TOTAL_SPANS || '10000');
const spanBytes = Number(process.env.SPAN_BYTES || '2000');

// Create payload
const attrPayload = 'x'.repeat(spanBytes);

// Log startup information
console.log(`[node-fixed-span-gen] Starting with ${totalSpans} total spans, ${spanBytes} bytes per span`);
console.log(`[node-fixed-span-gen] OTEL_SERVICE_NAME: ${process.env.OTEL_SERVICE_NAME || 'Not set'}`);
console.log(`[node-fixed-span-gen] OTEL_RESOURCE_ATTRIBUTES: ${process.env.OTEL_RESOURCE_ATTRIBUTES || 'Not set'}`);

function generateSpan(spanNum) {
  const span = tracer.startSpan('fixed-span');
  span.setAttribute('payload', attrPayload);
  span.setAttribute('lang', 'node');
  span.setAttribute('gen', 'node-fixed-span-gen');
  span.setAttribute('payload_size', spanBytes);
  span.setAttribute('span_number', spanNum);
  span.setAttribute('operation.type', 'fixed-load-test');
  span.setAttribute('user.id', spanNum % 10000);
  span.setAttribute('request.id', `req-${spanNum}-${Date.now()}`);
  span.setAttribute('trace.sampled', true);
  span.setAttribute('service.version', '1.0.0');
  span.setAttribute('deployment.environment', 'fixed-span-test');
  span.end();
}

async function main() {
  const startTime = Date.now();
  let generatedSpans = 0;

  console.log(`[node-fixed-span-gen] Starting to generate ${totalSpans} spans...`);

  // Generate all spans
  for (let i = 0; i < totalSpans; i++) {
    generateSpan(i + 1);
    generatedSpans++;

    // Log progress every 1000 spans
    if (generatedSpans % 1000 === 0) {
      const progress = (generatedSpans / totalSpans) * 100;
      console.log(`[node-fixed-span-gen] Generated ${generatedSpans}/${totalSpans} spans (${progress.toFixed(1)}%)`);
    }
  }

  const elapsed = (Date.now() - startTime) / 1000;
  const spansPerSec = totalSpans / elapsed;
  
  console.log(`[node-fixed-span-gen] Completed generating ${totalSpans} spans in ${elapsed.toFixed(2)} seconds (${spansPerSec.toFixed(2)} spans/sec)`);
  
  // Keep the container running for a bit to ensure all spans are exported
  console.log('[node-fixed-span-gen] Waiting 30 seconds to ensure all spans are exported...');
  await new Promise(resolve => setTimeout(resolve, 30000));
  
  console.log('[node-fixed-span-gen] Node.js fixed span generator completed successfully!');
}

main().catch(console.error);
