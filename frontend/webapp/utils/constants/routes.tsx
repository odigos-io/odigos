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

const HAS_WINDOW = typeof window !== 'undefined';
const IS_PROD = process.env.NODE_ENV === 'production';
const PORT = process.env.PORT || 8085;

const HTTP_ORIGIN = IS_PROD && HAS_WINDOW ? `http://${window.location.hostname}` : `http://localhost:${PORT}`;

export const API = {
  GRAPHQL: `${HTTP_ORIGIN}/graphql`,
  EVENTS: `${HTTP_ORIGIN}/api/events`,
};
