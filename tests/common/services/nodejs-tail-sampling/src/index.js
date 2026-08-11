var express = require('express');
var http = require('http');

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

function registerHopsRoute(path, options) {
  var propagateError = options.propagateError;

  app.get(path, function (req, res) {
    var hops = parseHops(req);
    var isError = shouldReturnError(req);
    var statusCode = propagateError
      ? (isError ? 500 : 200)
      : (hops === 1 && isError ? 500 : 200);

    if (hops === 1) {
      return res.status(statusCode).json({
        endpoint: path,
        description: options.finalHopDescription,
        hops_remaining: hops,
        forced_error: isError,
        status_code: statusCode,
        error_propagates_to_client: propagateError
      });
    }

    var nextPath = path + '?hops=' + (hops - 1) + (isError ? '&error=true' : '');
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
        res.status(statusCode).json({
          endpoint: path,
          description: options.hopDescription,
          hops_remaining: hops,
          next_path: nextPath,
          forced_error: isError,
          status_code: statusCode,
          error_propagates_to_client: propagateError,
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

function registerTailSamplingScenarioRoutes() {
  app.get('/sampling/tail/no-rule', function (req, res) {
    sendScenarioResponse(req, res, {
      endpoint: '/sampling/tail/no-rule',
      description: 'baseline traffic sampled through the 10% cost-reduction rule'
    });
  });

  app.get('/sampling/tail/error', function (req, res) {
    sendScenarioResponse(req, res, {
      endpoint: '/sampling/tail/error',
      description: 'tail-sampling error scenario; add ?error=true to force HTTP 500'
    });
  });

  app.get('/sampling/tail/duration/short', function (req, res) {
    sendScenarioResponse(req, res, {
      endpoint: '/sampling/tail/duration/short',
      description: 'short request duration, sampled through the 10% cost-reduction rule',
      delayMs: DURATIONS.short
    });
  });

  app.get('/sampling/tail/duration/medium', function (req, res) {
    sendScenarioResponse(req, res, {
      endpoint: '/sampling/tail/duration/medium',
      description: 'medium request duration above 500ms, sampled at 50%',
      delayMs: DURATIONS.medium
    });
  });

  app.get('/sampling/tail/duration/long', function (req, res) {
    sendScenarioResponse(req, res, {
      endpoint: '/sampling/tail/duration/long',
      description: 'long request duration above 1000ms, sampled at 100%',
      delayMs: DURATIONS.long
    });
  });

  registerHopsRoute('/sampling/tail/hops', {
    propagateError: true,
    finalHopDescription: 'final hop returns success or error based on the error query parameter',
    hopDescription: 'hop made an outgoing HTTP request to itself; downstream status is returned to the client'
  });

  registerHopsRoute('/sampling/tail/hops/non-propagating-error', {
    propagateError: false,
    finalHopDescription: 'final hop returns success or error based on the error query parameter',
    hopDescription: 'hop made an outgoing HTTP request to itself; only the final hop reflects a forced error on the client response'
  });
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

registerTailSamplingScenarioRoutes();
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
