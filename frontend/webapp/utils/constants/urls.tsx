const env = process.env.NODE_ENV;

const LOCALHOST = 'http://localhost:8085/api';
const BASE_URL = env === 'production' ? '/api' : LOCALHOST;

const API = {
  EVENTS: `${BASE_URL}/events`,
};

export const DOCS_LINK = 'https://docs.odigos.io';

export { API, BASE_URL };
