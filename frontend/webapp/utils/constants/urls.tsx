'use client';
const IS_PROD = process.env.NODE_ENV === 'production';

// set base URLs for all environments
const DEV_API_URL = 'http://localhost:8085';
const PROD_API_URL = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:3000';

// construct final base URL based on environment
const API_BASE_URL = IS_PROD ? PROD_API_URL : DEV_API_URL;

// add paths to base URL
const API = {
  GRAPHQL: `${API_BASE_URL}/graphql`,
  EVENTS: `${API_BASE_URL}/api/events`,
};

const DOCS_LINK = 'https://docs.odigos.io';

export { API, DOCS_LINK };
