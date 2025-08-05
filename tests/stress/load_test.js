import http from 'k6/http';

export const options = {
  scenarios: {
    ramping_rps: {
      executor: 'ramping-arrival-rate',
      startRate: 10,              // initial RPS
      timeUnit: '1s',
      preAllocatedVUs: 100,       // reserve enough VUs
      maxVUs: 200,                // cap max VUs
      stages: [
        { target: 30, duration: '30s' },   // ramp to 30 RPS in 30s
        { target: 90, duration: '1m' },    // ramp to 90 RPS in 1m
        { target: 0, duration: '30s' },    // cool down
      ],
    },
  },
};

const frontendURL = __ENV.FRONTEND_URL || 'http://localhost:8080';
const minProductID = 1;
const maxProductID = 20;
const watchProductID = 12;
let currentProductID = minProductID;

function getNextProductID() {
  currentProductID++;
  if (currentProductID === watchProductID) currentProductID++;
  if (currentProductID > maxProductID) currentProductID = minProductID;
  return currentProductID;
}

export default function () {
  const pid = getNextProductID();

  // POST /buy
  const buyRes = http.post(`${frontendURL}/buy?id=${pid}`);
  if (buyRes.status !== 200) {
    console.error(`Buy failed for product ${pid} - Status: ${buyRes.status}`);
  }

  // GET /products
  const getRes = http.get(`${frontendURL}/products`);
  if (getRes.status !== 200) {
    console.error(`Get products failed - Status: ${getRes.status}`);
  }

  // No sleep! RPS is controlled by K6 itself now.
}
