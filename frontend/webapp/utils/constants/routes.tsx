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
  PIPELINE_COLLECTORS: '/pipeline-collectors',
};

export const SKIP_TO_SUMMERY_QUERY_PARAM = 'skipToSummary';

const PROTOCOL = typeof window !== 'undefined' ? window.location.protocol : 'http:';
const HOSTNAME = typeof window !== 'undefined' ? window.location.hostname : '';
const PORT = typeof window !== 'undefined' ? window.location.port : '';

const IS_INGRESSED_DOMAIN = !!HOSTNAME && HOSTNAME !== 'localhost' && PORT === '';
const IS_DEV = process.env.NODE_ENV === 'development';
const DEFAULT_BACKEND_HTTP_ORIGIN = 'http://localhost:8085';

// TODO: improve this
const BACKEND_HTTP_ORIGIN = typeof window !== 'undefined' ? (IS_INGRESSED_DOMAIN ? `${PROTOCOL}//${HOSTNAME}` : window.location.origin) : 'http://localhost:3000';

export const API = {
  BACKEND_HTTP_ORIGIN,
  GRAPHQL: `${BACKEND_HTTP_ORIGIN}/graphql`,
  EVENTS: `${BACKEND_HTTP_ORIGIN}/api/events`,
};
