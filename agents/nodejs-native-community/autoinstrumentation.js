try {
    // Get the major Node.js version using traditional variable assignment and parsing.
    var nodeVersion = process.versions.node.split('.');
    var major = parseInt(nodeVersion[0], 10);

    // Check for supported Node.js version
    if (major < 14) {
        console.error('Odigos: Unsupported Node.js version for OpenTelemetry auto-instrumentation');
    } else {
        // Import the necessary functions using traditional require syntax.
        var opentelemetryNode = require('@odigos/opentelemetry-node');
        var createNativeCommunitySpanProcessor = opentelemetryNode.createNativeCommunitySpanProcessor;
        var startOpenTelemetryAgent = opentelemetryNode.startOpenTelemetryAgent;

        // Retrieve environment variables.
        var opampServerHost = process.env.ODIGOS_OPAMP_SERVER_HOST;
        var instrumentationDeviceId = process.env.ODIGOS_INSTRUMENTATION_DEVICE_ID;

        // Create a span processor and start the OpenTelemetry agent.
        var spanProcessor = createNativeCommunitySpanProcessor();
        startOpenTelemetryAgent('odigos-native-community', instrumentationDeviceId, opampServerHost, spanProcessor);
    }    
} catch (e) {
    console.error('Odigos: Failed to load OpenTelemetry auto-instrumentation agent native-community', e);
}
