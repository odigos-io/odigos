export const ROUTES = {
  ROOT: '/',
  ONBOARDING: '/onboarding',
  OVERVIEW: '/overview',
  SOURCES: '/sources',
  DESTINATIONS: '/destinations',
  ACTIONS: '/actions',
  INSTRUMENTATION_RULES: '/instrumentation-rules',
  SERVICE_MAP: '/service-map',
  PIPELINE_COLLECTORS: '/pipeline-collectors',
};

export const SKIP_TO_SUMMERY_QUERY_PARAM = 'skipToSummary';

const IS_DEV = process.env.NODE_ENV === 'development';
const HAS_WINDOW = typeof window !== 'undefined';
const DEFAULT = 'http://localhost:8085';

const BACKEND_HTTP_ORIGIN = IS_DEV || !HAS_WINDOW ? DEFAULT : window.location.origin;

export const API = {
  BACKEND_HTTP_ORIGIN,
  GRAPHQL: `${BACKEND_HTTP_ORIGIN}/graphql`,
  EVENTS: `${BACKEND_HTTP_ORIGIN}/api/events`,
};
