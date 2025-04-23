export const ROUTES = {
  ROOT: '/',
  CHOOSE_SOURCES: '/choose-sources',
  CHOOSE_DESTINATION: '/choose-destination',
  OVERVIEW: '/overview',
};

export const CRD_NAMES = {
  SOURCE: 'sources.odigos.io',
  INSTRUMENTATION_CONFIG: 'instrumentationconfigs.odigos.io',
  DESTINATION: 'destinations.odigos.io',
  ACTIONS: [
    'k8sattributesresolvers.actions.odigos.io',
    'addclusterinfos.actions.odigos.io',
    'deleteattributes.actions.odigos.io',
    'renameattributes.actions.odigos.io',
    'errorsamplers.actions.odigos.io',
    'latencysamplers.actions.odigos.io',
    'probabilisticsamplers.actions.odigos.io',
    'piimaskings.actions.odigos.io',
  ],
  INSTRUMENTATION_RULE: 'instrumentationrules.odigos.io',
};

export const NAMESPACES = {
  DEFAULT: 'default',
  ODIGOS_SYSTEM: 'odigos-system',
};

export const SELECTED_ENTITIES = {
  NAMESPACE: NAMESPACES.DEFAULT,
  NAMESPACE_SOURCES: ['coupon', 'currency', 'frontend', 'inventory', 'membership', 'pricing'],
  DESTINATION: {
    TYPE: 'jaeger',
    DISPLAY_NAME: 'Jaeger',
    AUTOFILL_FIELD: 'JAEGER_URL',
    AUTOFILL_VALUE: 'jaeger.tracing:4317',
  },
  ACTIONS: ['K8sAttributesResolver', 'AddClusterInfo', 'DeleteAttribute', 'RenameAttribute', 'ErrorSampler', 'LatencySampler', 'ProbabilisticSampler', 'PiiMasking'],
  INSTRUMENTATION_RULES: ['PayloadCollection', 'CodeAttributes'],
};

export const DATA_IDS = {
  SELECT_NAMESPACE: `[data-id=namespace-${SELECTED_ENTITIES.NAMESPACE}]`,
  SELECT_SOURCE: (sourceName: string) => `[data-id=source-${sourceName}]`,
  SELECT_DESTINATION: `[data-id=select-potential-destination-${SELECTED_ENTITIES.DESTINATION.TYPE}]`,
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
  SOURCE_NODE: (index: number) => `[data-id=Source-${index}]`,
  DESTINATION_NODE: (index: number) => `[data-id=Destination-${index}]`,
  ACTION_NODE: (index: number) => `[data-id=Action-${index}]`,
  INSTRUMENTATION_RULE_NODE: (index: number) => `[data-id=InstrumentationRule-${index}]`,

  TITLE: '[data-id=title]',
  SOURCE_TITLE: '[data-id=sourceName]',
  CHECKBOX: '[data-id=checkbox]',

  NOTIF_MANAGER_BUTTON: '[data-id=notif-manager-button]',
  NOTIF_MANAGER_CONTENR: '[data-id=notif-manager-content]',
};

export const BUTTONS = {
  BACK: 'BACK',
  NEXT: 'NEXT',
  DONE: 'DONE',
  ADD_DESTINATION: 'ADD Destination',
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

  NOTIF_DESTINATION_CREATED: (destinationType: string) => `Successfully created "${destinationType}" destination`,
  NOTIF_DESTINATION_UPDATED: (destinationType: string) => `Successfully updated "${destinationType}" destination`,
  NOTIF_DESTINATION_DELETED: (destinationType: string) => `Successfully deleted "${destinationType}" destination`,

  NOTIF_ACTION_CREATED: (actionType: string) => `Successfully created "${actionType}" action`,
  NOTIF_ACTION_UPDATED: (actionType: string) => `Successfully updated "${actionType}" action`,
  NOTIF_ACTION_DELETED: (actionType: string) => `Successfully deleted "${actionType}" action`,

  NOTIF_INSTRUMENTATION_RULE_CREATED: (ruleType: string) => `Successfully created "${ruleType}" rule`,
  NOTIF_INSTRUMENTATION_RULE_UPDATED: (ruleType: string) => `Successfully updated "${ruleType}" rule`,
  NOTIF_INSTRUMENTATION_RULE_DELETED: (ruleType: string) => `Successfully deleted "${ruleType}" rule`,
};
