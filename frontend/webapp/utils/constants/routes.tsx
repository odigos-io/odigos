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
  SERVICE_MAP: '/service-map',
};

export const SKIP_TO_SUMMERY_QUERY_PARAM = 'skipToSummary';

const PROTOCOL = typeof window !== 'undefined' ? window.location.protocol : 'http';
const HOSTNAME = typeof window !== 'undefined' ? window.location.hostname : '';
const IS_INGRESSED_DOMAIN = HOSTNAME && HOSTNAME !== 'localhost';

const BACKEND_PORT = process.env.PORT || 8085;
const BACKEND_HTTP_ORIGIN = IS_INGRESSED_DOMAIN ? `${PROTOCOL}//${HOSTNAME}` : `http://localhost:${BACKEND_PORT}`;

export const API = {
  GRAPHQL: `${BACKEND_HTTP_ORIGIN}/graphql`,
  EVENTS: `${BACKEND_HTTP_ORIGIN}/api/events`,
};
