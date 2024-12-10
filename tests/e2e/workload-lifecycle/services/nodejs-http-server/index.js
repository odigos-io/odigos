// verify the script set with --require flag is executed before the index.js file.
if(process.env.CHECK_FOR_APP_REQUIRE) {
  if(global.executeBeforeApplied !== true) {
    console.log('execute_before script was not executed before the index.js script');
    process.exit(1);
  }
}

if(process.env.CHECK_FOR_HEAP_SIZE) {
  const v8 = require('v8');

  // Get the total heap size limit in bytes - some workloads enable NODE_OPTIONS=--max-old-space-size=4096
  // But v8 only exposes heap_size_limit, which is the total heap size limit, including new space, old space, and code space.
  // we will check that the v8 heap size is around the expected value
  const heapSizeLimit = v8.getHeapStatistics().heap_size_limit;
  const heapSizeLimitInMB = heapSizeLimit / 1024 / 1024;
  const expectedHeapSizeMbFromEnv = 256
  if (heapSizeLimitInMB < expectedHeapSizeMbFromEnv || heapSizeLimitInMB >= expectedHeapSizeMbFromEnv * 2) {
    console.log(`Heap size limit is ${heapSizeLimitInMB} MB, expected a bit above ${expectedHeapSizeMbFromEnv} MB`);
    process.exit(1);
  }
}

// Import the express module
var express = require('express');

// Create an instance of the express application
var app = express();

// Define a route for the root URL that sends "Hello, World!" as the response
app.get('/', function (req, res) {
  res.send('Hello, World!');
});

// Start the server and listen on port 3000
app.listen(3000, function () {
  console.log('Server running at http://127.0.0.1:3000/');
});
