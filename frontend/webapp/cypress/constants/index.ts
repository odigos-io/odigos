export const ROUTES = {
  ROOT: '/',
  ONBOARDING: '/onboarding',
  OVERVIEW: '/overview',
  SETTINGS: '/settings',
};

export const CRD_NAMES = {
  SOURCE: 'sources.odigos.io',
  INSTRUMENTATION_CONFIG: 'instrumentationconfigs.odigos.io',
  DESTINATION: 'destinations.odigos.io',
  ACTION: 'actions.odigos.io',
  INSTRUMENTATION_RULE: 'instrumentationrules.odigos.io',
};

export const CONFIG_MAPS = {
  LOCAL_UI_CONFIG: 'odigos-local-ui-config',
  EFFECTIVE_CONFIG: 'effective-config',
};

export const NAMESPACES = {
  ODIGOS: 'odigos-test',
  APPS: 'default',
  DESTINATIONS: 'tracing',
};

export const SELECTED_ENTITIES = {
  NAMESPACE: NAMESPACES.APPS,
  NAMESPACE_SOURCES: [
    {
      namespace: NAMESPACES.APPS,
      name: 'coupon',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.APPS,
      name: 'currency',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.APPS,
      name: 'frontend',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.APPS,
      name: 'geolocation',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.APPS,
      name: 'inventory',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.APPS,
      name: 'membership',
      kind: 'Deployment',
    },
    {
      namespace: NAMESPACES.APPS,
      name: 'pricing',
      kind: 'Deployment',
    },
  ],
  DESTINATION: {
    TYPE: 'jaeger',
    DISPLAY_NAME: 'Jaeger',
    AUTOFILL_FIELD: 'JAEGER_URL',
    AUTOFILL_VALUE: `jaeger.${NAMESPACES.DESTINATIONS}:4317`,
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
    // TODO: uncomment when we fix data-ids for dropdown in ui-kit
    // 'SpanAttributeSampler',
  ],
  INSTRUMENTATION_RULES: ['PayloadCollection', 'CodeAttributes'],
};

export const DATA_IDS = {
  ONBOARDING_GET_STARTED: '[data-id=onboarding-get-started]',

  // v2 add-drawer selectors (sources)
  SELECT_NAMESPACE: `[data-id=namespace-${SELECTED_ENTITIES.NAMESPACE}]`,
  SELECT_SOURCE: (sourceName: string) => `[data-id=source-${sourceName}]`,

  // v2 add-drawer selectors (destinations)
  SELECT_DESTINATION: `[data-id="list-item-${SELECTED_ENTITIES.DESTINATION.DISPLAY_NAME}"]`,
  SELECT_DESTINATION_AUTOFILL_FIELD: `[name=${SELECTED_ENTITIES.DESTINATION.AUTOFILL_FIELD}]`,
  DEST_FORM_ADD: '[data-id=dest-form-add]',

  // v2 add-drawer selectors (actions & rules)
  ACTION_OPTION: (type: string) => `[data-id=option-${type}]`,
  RULE_OPTION: (type: string) => `[data-id=option-${type}]`,

  // data-flow "add" buttons (trigger drawer open)
  ADD_SOURCE: '[data-id=add-Source]',
  ADD_DESTINATION: '[data-id=add-Destination]',
  ADD_ACTION: '[data-id=add-Action]',
  ADD_INSTRUMENTATION_RULE: '[data-id=add-InstrumentationRule]',

  // legacy modals & edit-drawers
  MODAL: '[data-id=modal]',
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

  // v2 wide-drawer buttons
  WIDE_DRAWER_BACK: '[data-id=wide-drawer-back]',
  WIDE_DRAWER_NEXT: '[data-id=wide-drawer-next]',
  WIDE_DRAWER_SKIP: '[data-id=wide-drawer-skip]',
  WIDE_DRAWER_SAVE: '[data-id=wide-drawer-save]',
  WIDE_DRAWER_CANCEL: '[data-id=wide-drawer-cancel]',
  LIST_ITEM: (title: string) => `[data-id="list-item-${title}"]`,

  SETTINGS_SAVE: '[data-id=settings-save]',
  SETTINGS_CANCEL: '[data-id=settings-cancel]',
  SETTINGS_FIELD: (helmPath: string) => `[data-id="${helmPath}"]`,
};

export const BUTTONS = {
  BACK: 'BACK',
  NEXT: 'NEXT',
  DONE: 'DONE',
  ADD_DESTINATION: 'Add Destination',
  UNINSTRUMENT: 'Uninstrument',
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

  NOTIF_SOURCES_UPDATED: (name: string) => `Successfully updated "${name}" source`,

  NOTIF_DESTINATION_CREATED: (amount: number) => `Successfully created ${amount} destinations`,
  NOTIF_DESTINATION_UPDATED: (type: string) => `Successfully updated "${type}" destination`,
  NOTIF_DESTINATION_DELETED: (amount: number) => `Successfully deleted ${amount} destinations`,

  NOTIF_ACTION_CREATED: (actionType: string) => `Successfully created "${actionType}" action`,
  NOTIF_ACTION_UPDATED: (actionType: string) => `Successfully updated "${actionType}" action`,
  NOTIF_ACTION_DELETED: (actionType: string) => `Successfully deleted "${actionType}" action`,

  NOTIF_INSTRUMENTATION_RULE_CREATED: (ruleType: string) => `Successfully created "${ruleType}" rule`,
  NOTIF_INSTRUMENTATION_RULE_UPDATED: (ruleType: string) => `Successfully updated "${ruleType}" rule`,
  NOTIF_INSTRUMENTATION_RULE_DELETED: (ruleType: string) => `Successfully deleted "${ruleType}" rule`,

  NOTIF_CONFIG_UPDATED: 'Local UI configuration updated successfully',
};
