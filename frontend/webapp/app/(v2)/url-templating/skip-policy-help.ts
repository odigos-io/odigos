/** Help copy for skip-policy UI (see odigosurltemplateprocessor README). */
export const SKIP_POLICY_HELP = {
  title: 'Skip default templatization on HTTP errors',
  text: 'For internet-exposed services, bot and scanner traffic often hits random paths that return error status codes (for example 404). Default heuristic templatization would turn each of those into a distinct route and cause high cardinality. Custom templates are still evaluated first; skip policy only affects the default heuristic step when no custom rule matched. Use “Skip for non-2xx” to skip any status outside 2xx, or list specific codes such as 404 or 401.',
} as const;
