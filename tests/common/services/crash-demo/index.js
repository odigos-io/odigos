const http = require('http');

const PORT = 3000;
let requestCount = 0;

// Check if we're instrumented by looking for Odigos/OTel environment variables
function checkInstrumentation() {
  const otelVars = [
    'OTEL_SERVICE_NAME',
  ];

  return otelVars.some(varName => process.env[varName]);
}

// Function to crash the application when instrumentation is detected
function crashOnInstrumentation() {
  console.log('💥 INSTRUMENTATION DETECTED - CRASHING IMMEDIATELY!');
  console.log('📊 Environment variables found:');
  Object.keys(process.env)
    .filter(key => key.startsWith('OTEL_'))
    .forEach(key => console.log(`   ${key}=${process.env[key]}`));

  console.log('💀 This application is incompatible with OpenTelemetry instrumentation');
  console.log('🔄 Expecting Odigos auto-rollback to uninstrument this service...');

  // Exit immediately to simulate instrumentation incompatibility
  process.exit(1);
}

// Create HTTP server
const server = http.createServer((req, res) => {
  requestCount++;

  // Normal successful response (service only runs when NOT instrumented)
  const response = {
    message: 'Hello from crash demo service!',
    requestCount,
    instrumented: false,
    timestamp: new Date().toISOString(),
    status: 'healthy - no instrumentation detected'
  };

  res.writeHead(200, {
    'Content-Type': 'application/json',
    'X-Request-Count': requestCount.toString(),
    'X-Instrumented': 'false'
  });
  res.end(JSON.stringify(response, null, 2) + '\n');

  // Log request info
  console.log(`📝 Request ${requestCount}: ${req.method} ${req.url} - clean (no instrumentation)`);
});

// Graceful shutdown handling
process.on('SIGTERM', () => {
  console.log('🛑 SIGTERM received, shutting down gracefully...');
  server.close(() => {
    console.log('✅ Server closed');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('🛑 SIGINT received, shutting down gracefully...');
  server.close(() => {
    console.log('✅ Server closed');
    process.exit(0);
  });
});

// Start server
server.listen(PORT, () => {
  console.log(`🚀 Crash Demo Service started on port ${PORT}`);
  console.log(`📊 Process ID: ${process.pid}`);
  console.log(`🔍 Checking for instrumentation at startup...`);

  // Check for instrumentation immediately at startup
  if (checkInstrumentation()) {
    console.log('⚠️  Instrumentation detected at startup!');
    crashOnInstrumentation();
  } else {
    console.log('✅ No instrumentation detected - service running normally');
  }
});

// Handle uncaught exceptions
process.on('uncaughtException', (err) => {
  console.error('💀 Uncaught Exception:', err);
  process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
  console.error('💀 Unhandled Rejection at:', promise, 'reason:', reason);
  process.exit(1);
});
