var express = require('express');
var http = require('http');

console.log('nodejs-tail-sampling server starting...');

var app = express();
var PORT = 8080;

var DURATIONS = {
  short: 50,
  medium: 750,
  long: 1500
};

function shouldReturnError(req) {
  return req.query.error === 'true' || req.query.error === '1';
}

function sendScenarioResponse(req, res, scenario) {
  var isError = shouldReturnError(req);
  var statusCode = isError ? 500 : 200;
  var delayMs = scenario.delayMs || 0;

  setTimeout(function () {
    res.status(statusCode).json({
      endpoint: scenario.endpoint,
      description: scenario.description,
      simulated_duration_ms: delayMs,
      forced_error: isError,
      status_code: statusCode
    });
  }, delayMs);
}

function parseHops(req) {
  var hops = parseInt(req.query.hops, 10);
  if (Number.isNaN(hops) || hops < 1) {
    return 1;
  }
  return hops;
}

function parseDelayMs(req) {
  var ms = parseInt(req.query.ms, 10);
  if (Number.isNaN(ms) || ms < 0) {
    return 0;
  }
  return ms;
}

var alternateErrorNext = false;

function registerTailErrorScenarioRoutes() {
  app.get('/ok', function (req, res) {
    res.status(200).json({
      endpoint: '/ok',
      description: 'successful baseline request for cost-reduction tail sampling'
    });
  });

  app.get('/error', function (req, res) {
    res.status(500).json({
      endpoint: '/error',
      description: 'handler always returns HTTP 500 for error-focused tail sampling'
    });
  });

  app.get('/alternate', function (req, res) {
    var isError = alternateErrorNext;
    alternateErrorNext = !alternateErrorNext;
    var statusCode = isError ? 500 : 200;
    res.status(statusCode).json({
      endpoint: '/alternate',
      description: 'alternates HTTP 200 and 500 on each request (in-process toggle)',
      returned_error: isError,
      status_code: statusCode
    });
  });

  registerInternalErrorHopsRoute('/hops');
}

function registerTailDurationScenarioRoutes() {
  app.get('/duration', function (req, res) {
    var delayMs = parseDelayMs(req);
    sendScenarioResponse(req, res, {
      endpoint: '/duration',
      description: 'response delayed by ?ms= query parameter',
      delayMs: delayMs
    });
  });

  app.get('/duration/short', function (req, res) {
    sendScenarioResponse(req, res, {
      endpoint: '/duration/short',
      description: 'short request duration (~50ms), sampled through the 10% cost-reduction rule',
      delayMs: DURATIONS.short
    });
  });

  app.get('/duration/medium', function (req, res) {
    sendScenarioResponse(req, res, {
      endpoint: '/duration/medium',
      description: 'medium request duration (~750ms), sampled at least 50%',
      delayMs: DURATIONS.medium
    });
  });

  app.get('/duration/long', function (req, res) {
    sendScenarioResponse(req, res, {
      endpoint: '/duration/long',
      description: 'long request duration (~1500ms), sampled at 100%',
      delayMs: DURATIONS.long
    });
  });
}

function registerInternalErrorHopsRoute(path) {
  app.get(path, function (req, res) {
    var hops = parseHops(req);

    if (hops === 1) {
      return res.status(500).json({
        endpoint: path,
        description: 'final hop always returns HTTP 500 (error on internal span only)',
        hops_remaining: hops,
        status_code: 500
      });
    }

    var nextPath = path + '?hops=' + (hops - 1);
    http.get({
      hostname: '127.0.0.1',
      port: PORT,
      path: nextPath
    }, function (selfRes) {
      var body = '';

      selfRes.setEncoding('utf8');
      selfRes.on('data', function (chunk) {
        body += chunk;
      });
      selfRes.on('end', function () {
        res.status(200).json({
          endpoint: path,
          description: 'self HTTP hop; last hop is always 500, caller always gets HTTP 200',
          hops_remaining: hops,
          next_path: nextPath,
          status_code: 200,
          downstream_status_code: selfRes.statusCode,
          downstream_body: body
        });
      });
    }).on('error', function (err) {
      res.status(502).json({
        endpoint: path,
        error: err.message
      });
    });
  });
}

app.get('/healthz', function (req, res) {
  res.status(200).json({ status: 'healthy' });
});

// Express path params so http.route is templated
// (e.g. /http-server/templated/:id/users/:name/orders/:uuid) rather than the concrete URL.
// Callers should vary id/name/uuid per request.
app.get('/http-server/templated/:id/users/:name/orders/:uuid', function (req, res) {
  res.status(200).json({
    endpoint: '/http-server/templated/:id/users/:name/orders/:uuid',
    id: req.params.id,
    name: req.params.name,
    uuid: req.params.uuid,
    description: 'templated path with id, name, and uuid segments for http.route asserts'
  });
});

// Templated prefix: nested /items/:itemId also matches a routePrefix rule ending at /items.
app.get('/http-server/prefix/:id/items', function (req, res) {
  res.status(200).json({
    endpoint: '/http-server/prefix/:id/items',
    id: req.params.id,
    description: 'templated prefix path for routePrefix sampling'
  });
});

app.get('/http-server/prefix/:id/items/:itemId', function (req, res) {
  res.status(200).json({
    endpoint: '/http-server/prefix/:id/items/:itemId',
    id: req.params.id,
    itemId: req.params.itemId,
    description: 'nested path under templated prefix for routePrefix sampling'
  });
});

// Static paths (no :params) so sampling AllStatic rules match http.route exactly.
app.get('/http-server/static/leading-slash', function (req, res) {
  res.status(200).json({
    endpoint: '/http-server/static/leading-slash',
    description: 'static path sampled by a rule whose route starts with /'
  });
});

app.get('/http-server/static/no-leading-slash', function (req, res) {
  res.status(200).json({
    endpoint: '/http-server/static/no-leading-slash',
    description: 'static path sampled by a rule whose route has no leading /'
  });
});

registerTailErrorScenarioRoutes();
registerTailDurationScenarioRoutes();

var server = app.listen(PORT, function () {
  console.log('tail-sampling server running at http://127.0.0.1:' + PORT + '/');
});

process.on('SIGTERM', function () {
  console.log('SIGTERM received, shutting down gracefully...');
  server.close(function () {
    console.log('HTTP server closed');
    process.exit(0);
  });
  setTimeout(function () {
    console.error('Could not close connections in time, forcefully shutting down');
    process.exit(1);
  }, 10000);
});
