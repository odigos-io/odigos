export const ROUTES = {
  ROOT: '/',
  CHOOSE_STREAM: '/choose-stream',
  CHOOSE_SOURCES: '/choose-sources',
  CHOOSE_DESTINATION: '/choose-destination',
  SETUP_SUMMARY: '/setup-summary',
  OVERVIEW: '/overview',
};

export const CRD_NAMES = {
  SOURCE: 'sources.odigos.io',
  INSTRUMENTATION_CONFIG: 'instrumentationconfigs.odigos.io',
  DESTINATION: 'destinations.odigos.io',
  ACTION: 'actions.odigos.io',
  INSTRUMENTATION_RULE: 'instrumentationrules.odigos.io',
};

export const NAMESPACES = {
  DEFAULT: 'default',
  ODIGOS_SYSTEM: 'odigos-system',
  ODIGOS_TEST: 'odigos-test',
};

export const SELECTED_ENTITIES = {
  NAMESPACE: NAMESPACES.DEFAULT,
  NAMESPACE_SOURCES: [
    {
      namespace: NAMESPACES.DEFAULT,
      name: 'coupon',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.DEFAULT,
      name: 'currency',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.DEFAULT,
      name: 'frontend',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.DEFAULT,
      name: 'geolocation',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.DEFAULT,
      name: 'inventory',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.DEFAULT,
      name: 'membership',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.DEFAULT,
      name: 'pricing',
      kind: 'Deployment',
    },
  ],
  DESTINATION: {
    TYPE: 'jaeger',
    DISPLAY_NAME: 'Jaeger',
    AUTOFILL_FIELD: 'JAEGER_URL',
    AUTOFILL_VALUE: 'jaeger.tracing:4317',
  },
  ACTIONS: [
    'K8sAttributesResolver',
    'AddClusterInfo',
    'DeleteAttribute',
    'RenameAttribute',
    'PiiMasking',
    'ErrorSampler',
    'LatencySampler',
    'ProbabilisticSampler',
    'ServiceNameSampler',
    'SpanAttributeSampler',
  ],
  INSTRUMENTATION_RULES: ['PayloadCollection', 'CodeAttributes'],
};

export const DATA_IDS = {
  SELECT_NAMESPACE: `[data-id=namespace-${SELECTED_ENTITIES.NAMESPACE}]`,
  SELECT_SOURCE: (sourceName: string) => `[data-id=source-${sourceName}]`,
  SELECT_DESTINATION: `[data-id=select-detectedbyodigos-destination-${SELECTED_ENTITIES.DESTINATION.TYPE}]`,
  SELECT_DESTINATION_AUTOFILL_FIELD: `[data-id=${SELECTED_ENTITIES.DESTINATION.AUTOFILL_FIELD}]`,

  ADD_SOURCE: '[data-id=add-Source]',
  ADD_DESTINATION: '[data-id=add-Destination]',
  ADD_ACTION: '[data-id=add-Action]',
  ADD_INSTRUMENTATION_RULE: '[data-id=add-InstrumentationRule]',

  MODAL: '[data-id=modal]',
  MODAL_ADD_SOURCE: '[data-id=modal-Add-Source]',
  MODAL_ADD_DESTINATION: '[data-id=modal-Add-Destination]',
  MODAL_ADD_ACTION: '[data-id=modal-Add-Action]',
  MODAL_ADD_INSTRUMENTATION_RULE: '[data-id=modal-Add-Instrumentation-Rule]',
  ACTION_OPTION: (type: string) => `[data-id=option-${type}]`,
  RULE_OPTION: (type: string) => `[data-id=option-${type}]`,

  DRAWER: '[data-id=drawer]',
  DRAWER_EDIT: '[data-id=drawer-edit]',
  DRAWER_SAVE: '[data-id=drawer-save]',
  DRAWER_CLOSE: '[data-id=drawer-close]',
  DRAWER_DELETE: '[data-id=drawer-delete]',
  APPROVE: '[data-id=approve]',
  DENY: '[data-id=deny]',

  TOAST: '[data-id=toast]',
  TOAST_CLOSE: '[data-id=toast-close]',
  TOAST_ACTION: '[data-id=toast-action]',

  MULTI_SOURCE_CONTROL: '[data-id=multi-source-control]',
  SOURCE_NODE_HEADER: '[data-id=Source-header]',
  SOURCE_NODE: (id: { namespace: string; name: string; kind: string }) => `[data-id=${id.namespace}-${id.name}-${id.kind}]`,
  DESTINATION_NODE: (id: string) => `[data-id="${id}"]`,

  TITLE: '[data-id=title]',
  SOURCE_TITLE: '[data-id=sourceName]',
  CHECKBOX: '[data-id=checkbox]',
};

export const BUTTONS = {
  BACK: 'BACK',
  NEXT: 'NEXT',
  DONE: 'DONE',
  ADD_DESTINATION: 'Add Destination',
  UNINSTRUMENT: 'Uninstrument',
};

export const INPUTS = {
  ACTION_DROPDOWN: 'Type to search...',
  RULE_DROPDOWN: 'Type to search...',
};

const CYPRESS_TEST = 'Cypress Test';

export const TEXTS = {
  UPDATED_NAME: CYPRESS_TEST,

  NO_RESOURCES: (namespace: string) => `No resources found in ${namespace} namespace.`,
  NO_SOURCES_SELECTED: 'No sources selected. Please go back to select sources.',

  SOURCE_WARN_MODAL_TITLE: (count: number) => `Uninstrument ${count} sources`,
  SOURCE_WARN_MODAL_NOTE: "You're about to uninstrument the last Source",
  DESTINATION_WARN_MODAL_TITLE: `Delete Destination (${CYPRESS_TEST})`,
  DESTINATION_WARN_MODAL_NOTE: "You're about to delete the last Destination",
  ACTION_WARN_MODAL_TITLE: `Delete Action (${CYPRESS_TEST})`,
  INSTRUMENTATION_RULE_WARN_MODAL_TITLE: `Delete InstrumentationRule (${CYPRESS_TEST})`,

  NOTIF_CREATED: 'Successfully created',
  NOTIF_UPDATED: 'Successfully updated',
  NOTIF_DELETED: 'Successfully deleted',

  NOTIF_SOURCES_PERSISTING: 'Persisting sources...',
  NOTIF_SOURCE_UPDATING: 'Updating source...',
  NOTIF_DESTINATION_UPDATING: 'Updating destination...',

  NOTIF_SOURCES_CREATED: (amount: number) => `Successfully created ${amount} sources`,
  NOTIF_SOURCES_UPDATED: (name: string) => `Successfully updated "${name}" source`,
  NOTIF_SOURCES_DELETED: (amount: number) => `Successfully deleted ${amount} sources`,

  NOTIF_DESTINATION_CREATED: (amount: number) => `Successfully created ${amount} destinations`,
  NOTIF_DESTINATION_UPDATED: (type: string) => `Successfully updated "${type}" destination`,
  NOTIF_DESTINATION_DELETED: (amount: number) => `Successfully deleted ${amount} destinations`,

  NOTIF_ACTION_CREATED: (actionType: string) => `Successfully created "${actionType}" action`,
  NOTIF_ACTION_UPDATED: (actionType: string) => `Successfully updated "${actionType}" action`,
  NOTIF_ACTION_DELETED: (actionType: string) => `Successfully deleted "${actionType}" action`,

  NOTIF_INSTRUMENTATION_RULE_CREATED: (ruleType: string) => `Successfully created "${ruleType}" rule`,
  NOTIF_INSTRUMENTATION_RULE_UPDATED: (ruleType: string) => `Successfully updated "${ruleType}" rule`,
  NOTIF_INSTRUMENTATION_RULE_DELETED: (ruleType: string) => `Successfully deleted "${ruleType}" rule`,
};
