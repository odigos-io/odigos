'use client';
const ENV = process.env.NODE_ENV;
const IS_PRODUCTION = ENV === 'production';

// Define base URLs depending on the environment and rendering context
const LOCAL_API_BASE = 'http://localhost:8085';
//we use localhost:8085 as the base URL for server environment
const PRODUCTION_GQL_API_BASE = IS_PRODUCTION && typeof window !== 'undefined' ? `${window.location.origin}/graphql` : `${LOCAL_API_BASE}/graphql`;
const API_BASE_URL = IS_PRODUCTION ? PRODUCTION_GQL_API_BASE : `${LOCAL_API_BASE}/graphql`;

// Define endpoints based on the base URL
const API = {
<<<<<<< HEAD
  EVENTS: `${BASE_URL}/events`,
};

export const DOCS_LINK = 'https://docs.odigos.io';

export { API, BASE_URL };
=======
  BASE_URL: API_BASE_URL,
  EVENTS: `${IS_PRODUCTION ? '/' : LOCAL_API_BASE}/api/events`,
};

// Centralize external links
export const DOCS_LINK = 'https://docs.odigos.io';

// Export modules
export { API };
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
