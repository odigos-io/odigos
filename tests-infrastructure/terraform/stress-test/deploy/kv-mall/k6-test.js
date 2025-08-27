// /opt/k6/tests/loadtest.js
import http from 'k6/http';

export const options = {
  scenarios: {
    ramping_rps: {
      executor: 'constant-arrival-rate',
      rate: 750000,
      timeUnit: '1s',
      duration: '60s',
      preAllocatedVUs: 100,
      maxVUs: 200
    },
  },
};

// Shared ALB DNS
const albURL = 'http://internal-k8s-loadtest-odigosag-224d31b749-815086470.us-east-1.elb.amazonaws.com';

// List of namespace paths
const namespaces = ['/kv-mall','/kv-mall-1', '/kv-mall-2', '/kv-mall-3'];

let currentProductID = 1;
const maxProductID = 20;

function getNextProductID() {
  currentProductID++;
  if (currentProductID > maxProductID) currentProductID = 1;

  if (currentProductID == 12) currentProductID++;
  return currentProductID;
}

export default function () {
  const ns = namespaces[Math.floor(Math.random() * namespaces.length)];
  const pid = getNextProductID();

  const buyRes = http.post(`${albURL}${ns}/buy?id=${pid}`);
  if (buyRes.status !== 200) {
    console.error(`Buy failed in ${ns} for product ${pid} - Status: ${buyRes.status}`);
  }

  const getRes = http.get(`${albURL}${ns}/products`);
  if (getRes.status !== 200) {
    console.error(`Get products failed in ${ns} - Status: ${getRes.status}`);
  }
}