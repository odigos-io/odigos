export const ROUTES = {
  ROOT: '/',
  CHOOSE_STREAM: '/choose-stream',
  CHOOSE_SOURCES: '/choose-sources',
  CHOOSE_DESTINATION: '/choose-destination',
  SETUP_SUMMARY: '/setup-summary',
  OVERVIEW: '/overview',
  SOURCES: '/sources',
  DESTINATIONS: '/destinations',
  ACTIONS: '/actions',
  INSTRUMENTATION_RULES: '/instrumentation-rules',
};

export const SKIP_TO_SUMMERY_QUERY_PARAM = 'skipToSummary';

const IS_PROD = process.env.NODE_ENV === 'production';

// set base URLs for all environments
const DEV_API_URL = 'http://localhost:8085';
const PROD_API_URL = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:3000';

// construct final base URL based on environment
const API_BASE_URL = IS_PROD ? PROD_API_URL : DEV_API_URL;

// add paths to base URL
export const API = {
  GRAPHQL: `${API_BASE_URL}/graphql`,
  EVENTS: `${API_BASE_URL}/api/events`,
};
