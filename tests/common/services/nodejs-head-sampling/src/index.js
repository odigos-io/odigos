var express = require('express');

var app = express();
var PORT = 8080;

var startTime = Date.now();
var SIMULATE_STARTUP_DELAY = process.env.SIMULATE_STARTUP_DELAY === 'true';
var STARTUP_DELAY_MS = 10000;
var READY_DELAY_MS = 12000;

app.get('/healthz/startup', function (req, res) {
  if (!SIMULATE_STARTUP_DELAY) {
    return res.status(200).json({ status: 'started', simulated: false });
  }
  var elapsed = Date.now() - startTime;
  if (elapsed >= STARTUP_DELAY_MS) {
    res.status(200).json({ status: 'started', elapsed_ms: elapsed });
  } else {
    res.status(503).json({ status: 'starting', remaining_ms: STARTUP_DELAY_MS - elapsed });
  }
});

app.get('/healthz/ready', function (req, res) {
  if (!SIMULATE_STARTUP_DELAY) {
    return res.status(200).json({ status: 'ready', simulated: false });
  }
  var elapsed = Date.now() - startTime;
  if (elapsed >= READY_DELAY_MS) {
    res.status(200).json({ status: 'ready', elapsed_ms: elapsed });
  } else {
    res.status(503).json({ status: 'not_ready', remaining_ms: READY_DELAY_MS - elapsed });
  }
});

app.get('/healthz/live', function (req, res) {
  res.status(200).json({ status: 'alive' });
});

app.get('/healthz', function (req, res) {
  res.status(200).json({ status: 'healthy' });
});

function resolveGraphqlHealthCheckProbeType(req) {
  var operationName = req.query.operationName;
  if (operationName === 'HealthCheckStartup') {
    return 'startup';
  }
  if (operationName === 'HealthCheckReady') {
    return 'ready';
  }
  if (operationName === 'HealthCheckLive') {
    return 'live';
  }

  var queryParam = req.query.query;
  var queries = Array.isArray(queryParam) ? queryParam : queryParam ? [queryParam] : [];
  if (queries.length >= 2) {
    return 'live';
  }

  var query = queries[0] || '';
  if (query.indexOf('HealthCheckStartup') !== -1) {
    return 'startup';
  }
  if (query.indexOf('HealthCheckReady') !== -1) {
    return 'ready';
  }
  if (query.indexOf('HealthCheckLive') !== -1) {
    return 'live';
  }
  return null;
}

function graphqlHealthCheckData(probeType) {
  if (probeType === 'live') {
    return { data: { status: 'alive', __typename: 'Query' } };
  }
  if (probeType === 'startup') {
    return { data: { status: 'started', __typename: 'Query' } };
  }
  return { data: { status: 'ready', __typename: 'Query' } };
}

function graphqlApplicativeData(req) {
  var operationName = req.query.operationName;
  var queryParam = req.query.query;
  var queries = Array.isArray(queryParam) ? queryParam : queryParam ? [queryParam] : [];
  var query = queries[0] || '';

  if (operationName === 'GetUser' || query.indexOf('GetUser') !== -1) {
    return { data: { user: { id: '1', name: 'Alice', __typename: 'User' }, __typename: 'Query' } };
  }
  if (operationName === 'ListItems' || query.indexOf('ListItems') !== -1) {
    return {
      data: {
        items: [{ id: '1', title: 'Item 1', __typename: 'Item' }],
        __typename: 'Query',
      },
    };
  }
  if (operationName === 'GetItem' || query.indexOf('GetItem') !== -1) {
    return { data: { item: { id: '42', title: 'Item 42', __typename: 'Item' }, __typename: 'Query' } };
  }
  return { data: { ok: true, __typename: 'Query' } };
}

app.get('/graphql', function (req, res) {
  var probeType = resolveGraphqlHealthCheckProbeType(req);
  if (!probeType) {
    return res.status(200).json(graphqlApplicativeData(req));
  }

  if (probeType === 'live') {
    return res.status(200).json(graphqlHealthCheckData('live'));
  }

  if (!SIMULATE_STARTUP_DELAY) {
    return res.status(200).json(graphqlHealthCheckData(probeType));
  }

  var elapsed = Date.now() - startTime;
  if (probeType === 'startup') {
    if (elapsed >= STARTUP_DELAY_MS) {
      return res.status(200).json(graphqlHealthCheckData('startup'));
    }
    return res.status(503).json({
      errors: [{ message: 'starting', extensions: { remaining_ms: STARTUP_DELAY_MS - elapsed } }],
    });
  }

  if (elapsed >= READY_DELAY_MS) {
    return res.status(200).json(graphqlHealthCheckData('ready'));
  }
  return res.status(503).json({
    errors: [{ message: 'not_ready', extensions: { remaining_ms: READY_DELAY_MS - elapsed } }],
  });
});

// No sampling rule targets this endpoint — traces are subject to the default pipeline behavior
app.get('/sampling/percentage/no-rule', function (req, res) {
  res.status(200).json({ endpoint: '/sampling/percentage/no-rule', description: 'no sampling rule matches this endpoint' });
});

// A sampling rule targets this endpoint with 0% sampling rate
app.get('/sampling/percentage/sampled-0', function (req, res) {
  res.status(200).json({ endpoint: '/sampling/percentage/sampled-0', description: 'sampling rule with 0% rate' });
});

// A sampling rule targets this endpoint with 50% sampling rate
app.get('/sampling/percentage/sampled-50', function (req, res) {
  res.status(200).json({ endpoint: '/sampling/percentage/sampled-50', description: 'sampling rule with 50% rate' });
});

// A sampling rule targets this endpoint with 100% sampling rate
app.get('/sampling/percentage/sampled-100', function (req, res) {
  res.status(200).json({ endpoint: '/sampling/percentage/sampled-100', description: 'sampling rule with 100% rate' });
});

// A sampling rule targets this endpoint with no percentage (fallback to default 0%)
app.get('/sampling/percentage/sampled-fallback', function (req, res) {
  res.status(200).json({ endpoint: '/sampling/percentage/sampled-fallback', description: 'sampling rule with no percentage, falls back to 0%' });
});

// A sampling rule targets this route prefix with 50% sampling rate
app.get('/sampling/route/prefix', function (req, res) {
  res.status(200).json({ endpoint: '/sampling/route/prefix', description: 'route prefix sampling rule with 50% rate' });
});

app.get('/sampling/route/prefix/part-one', function (req, res) {
  res.status(200).json({ endpoint: '/sampling/route/prefix/part-one', description: 'matches the /sampling/route/prefix sampling rule' });
});

app.get('/sampling/route/prefix/part-one/part-two', function (req, res) {
  res.status(200).json({ endpoint: '/sampling/route/prefix/part-one/part-two', description: 'matches the /sampling/route/prefix sampling rule with more route parts' });
});

// Sampling rules target these exact route templates, not the concrete request paths
app.get('/sampling/route/exact/:itemId', function (req, res) {
  res.status(200).json({ endpoint: '/sampling/route/exact/:itemId', item_id: req.params.itemId, description: 'exact sampling rule for a templated route' });
});

app.get('/sampling/route/exact/:itemId/details/:detailId', function (req, res) {
  res.status(200).json({ endpoint: '/sampling/route/exact/:itemId/details/:detailId', item_id: req.params.itemId, detail_id: req.params.detailId, description: 'exact sampling rule for a templated route with multiple parameters' });
});

// HTTP route matching scenarios (see deployments/http-server/ and deployments/http-client/)
function registerHttpMatchRoutes() {
app.get('/http-match/control/no-rule', function (req, res) {
  res.status(200).json({
    category: 'control',
    endpoint: '/http-match/control/no-rule',
    description: 'no sampling rule targets this path; traces follow default pipeline behavior',
  });
});

// Exact route match — literal path (sampling rule uses operation.httpServer.route)
app.get('/http-match/exact/target', function (req, res) {
  res.status(200).json({
    category: 'exact_route',
    endpoint: '/http-match/exact/target',
    description: 'exact HTTP route match (full literal path)',
  });
});

// Exact path for outbound POST tests (peer); not GET.
app.post('/http-match/exact/post-target', function (req, res) {
  res.status(200).json({
    category: 'exact_route_post',
    endpoint: '/http-match/exact/post-target',
    description: 'POST-only exact path for client outbound tests',
  });
});

// Prefix route match — static prefix; nested paths also match routePrefix rules
app.get('/http-match/prefix/segment', function (req, res) {
  res.status(200).json({
    category: 'prefix_route',
    endpoint: '/http-match/prefix/segment',
    description: 'prefix HTTP route match (first segment after /http-match/prefix)',
  });
});

app.get('/http-match/prefix/segment/nested', function (req, res) {
  res.status(200).json({
    category: 'prefix_route',
    endpoint: '/http-match/prefix/segment/nested',
    description: 'prefix HTTP route match (deeper path under the same static prefix)',
  });
});

// Templatized exact match — fixed pattern with parameterized segments (rule uses * in route)
app.get('/http-match/texact/:resourceId', function (req, res) {
  res.status(200).json({
    category: 'templatized_exact',
    endpoint: '/http-match/texact/:resourceId',
    resource_id: req.params.resourceId,
    description: 'templatized exact route (one dynamic segment)',
  });
});

// Templatized prefix match — wildcard segment in the prefix before a literal suffix
app.get('/http-match/tprefix/:tenantId/items', function (req, res) {
  res.status(200).json({
    category: 'templatized_prefix',
    endpoint: '/http-match/tprefix/:tenantId/items',
    tenant_id: req.params.tenantId,
    description: 'templatized prefix (tenant/items)',
  });
});

app.get('/http-match/tprefix/:tenantId/items/:itemId', function (req, res) {
  res.status(200).json({
    category: 'templatized_prefix',
    endpoint: '/http-match/tprefix/:tenantId/items/:itemId',
    tenant_id: req.params.tenantId,
    item_id: req.params.itemId,
    description: 'templatized prefix with extra path under items',
  });
});
}

if (process.env.HTTP_MATCH_ENABLE_ROUTES !== 'false') {
  registerHttpMatchRoutes();
}

// When HTTP_MATCH_PERIODIC_OUTBOUND=true (http-client deployment), issue periodic outbound HTTP client requests.
var httpMatchOutboundTimer = null;

function startHttpMatchPeriodicOutboundIfEnabled() {
  if (process.env.HTTP_MATCH_PERIODIC_OUTBOUND !== 'true') {
    return;
  }
  var defaultPeerBase = process.env.HTTP_MATCH_PEER_BASE_URL || 'http://127.0.0.1:' + PORT;
  var exactPeerBase =
    process.env.HTTP_MATCH_EXACT_PEER_BASE_URL || defaultPeerBase;
  var intervalMs = parseInt(process.env.HTTP_MATCH_OUTBOUND_INTERVAL_MS || '10000', 10);
  if (isNaN(intervalMs) || intervalMs < 1000) {
    intervalMs = 10000;
  }
  var exactGetPath = '/http-match/exact/target';
  var postExactPath = '/http-match/exact/post-target';
  var prefixAndTemplatedPaths = [
    '/http-match/prefix/segment',
    '/http-match/prefix/segment/nested',
    '/http-match/texact/out-peer-res',
    '/http-match/tprefix/out-peer-tenant/items',
    '/http-match/tprefix/out-peer-tenant/items/out-peer-item',
  ];
  var requestsPerTick = 2 + 1 + prefixAndTemplatedPaths.length;
  function fireOutbound() {
    var defaultBase = defaultPeerBase.replace(/\/$/, '');
    var exactBase = exactPeerBase.replace(/\/$/, '');
    fetch(defaultBase + exactGetPath).catch(function (err) {
      console.error(
        'http-match periodic outbound failed',
        defaultBase + exactGetPath,
        err.message
      );
    });
    fetch(exactBase + exactGetPath).catch(function (err) {
      console.error(
        'http-match periodic outbound failed',
        exactBase + exactGetPath,
        err.message
      );
    });
    fetch(defaultBase + postExactPath, { method: 'POST' }).catch(function (err) {
      console.error(
        'http-match periodic outbound POST failed',
        defaultBase + postExactPath,
        err.message
      );
    });
    prefixAndTemplatedPaths.forEach(function (relPath) {
      var url = defaultBase + relPath;
      fetch(url).catch(function (err) {
        console.error('http-match periodic outbound failed', url, err.message);
      });
    });
  }
  console.log(
    'http-match periodic outbound every ' +
      intervalMs +
      'ms → ' +
      requestsPerTick +
      ' requests/tick (GET exact ×2 + POST exact + prefix/templated GETs; bases ' +
      defaultPeerBase +
      ' / ' +
      exactPeerBase +
      ')'
  );
  httpMatchOutboundTimer = setInterval(fireOutbound, intervalMs);
}

var server = app.listen(PORT, function () {
  console.log('head-sampling server running at http://127.0.0.1:' + PORT + '/');
  if (SIMULATE_STARTUP_DELAY) {
    console.log('Startup probe will pass after ' + (STARTUP_DELAY_MS / 1000) + 's');
    console.log('Readiness probe will pass after ' + (READY_DELAY_MS / 1000) + 's');
  } else {
    console.log('Startup delay simulation is disabled');
  }
  startHttpMatchPeriodicOutboundIfEnabled();
});

process.on('SIGTERM', function () {
  console.log('SIGTERM received, shutting down gracefully...');
  if (httpMatchOutboundTimer) {
    clearInterval(httpMatchOutboundTimer);
    httpMatchOutboundTimer = null;
  }
  server.close(function () {
    console.log('HTTP server closed');
    process.exit(0);
  });
  setTimeout(function () {
    console.error('Could not close connections in time, forcefully shutting down');
    process.exit(1);
  }, 10000);
});
