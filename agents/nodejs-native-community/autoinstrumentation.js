try {
    const [major] = process.versions.node.split('.').map(Number);
    if (major < 14) {
      console.error('Odigos: Unsupported Node.js version for OpenTelemetry auto-instrumentation');
    } else {
        require('@odigos/opentelemetry-node')
    }    
} catch (e) {
    console.error('Odigos: Failed to load OpenTelemetry auto-instrumentation', e);
}
