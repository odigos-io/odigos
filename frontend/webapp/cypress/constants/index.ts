export const ROUTES = {
  ROOT: '/',
  CHOOSE_SOURCES: '/choose-sources',
  CHOOSE_DESTINATION: '/choose-destination',
  OVERVIEW: '/overview',
};

export const CRD_NAMES = {
  SOURCE: 'instrumentationconfigs.odigos.io',
  DESTINATION: 'destinations.odigos.io',
  ACTION: 'piimaskings.actions.odigos.io',
  INSTRUMENTATION_RULE: 'instrumentationrules.odigos.io',
};

export const CRD_IDS = {
  SOURCE: 'deployment-frontend',
  DESTINATION: '',
  ACTION: '',
  INSTRUMENTATION_RULE: '',
};

export const NAMESPACES = {
  DEFAULT: 'default',
  ODIGOS_SYSTEM: 'odigos-system',
};

export const SELECTED_ENTITIES = {
  NAMESPACE: NAMESPACES.DEFAULT,
  SOURCE: 'frontend',
  DESTINATION: 'Jaeger',
  DESTINATION_AUTOFILL_FIELD: 'JAEGER_URL',
  ACTION: 'PiiMasking',
  INSTRUMENTATION_RULE: 'PayloadCollection',
};

export const DATA_IDS = {
  SELECT_NAMESPACE: `[data-id=namespace-${SELECTED_ENTITIES.NAMESPACE}]`,
  SELECT_DESTINATION: `[data-id=destination-${SELECTED_ENTITIES.DESTINATION}]`,
  SELECT_DESTINATION_AUTOFILL_FIELD: `[data-id=${SELECTED_ENTITIES.DESTINATION_AUTOFILL_FIELD}]`,

  ADD_ENTITY: '[data-id=add-entity]',
  ADD_SOURCE: '[data-id=add-source]',
  ADD_DESTINATION: '[data-id=add-destination]',
  ADD_ACTION: '[data-id=add-action]',
  ADD_INSTRUMENTATION_RULE: '[data-id=add-rule]',

  MODAL: '[data-id=modal]',
  MODAL_ADD_SOURCE: '[data-id=modal-Add-Source]',
  MODAL_ADD_DESTINATION: '[data-id=modal-Add-Destination]',
  MODAL_ADD_ACTION: '[data-id=modal-Add-Action]',
  MODAL_ADD_INSTRUMENTATION_RULE: '[data-id=modal-Add-Instrumentation-Rule]',

  DRAWER: '[data-id=drawer]',
  DRAWER_EDIT: '[data-id=drawer-edit]',
  DRAWER_SAVE: '[data-id=drawer-save]',
  DRAWER_CLOSE: '[data-id=drawer-close]',
  DRAWER_DELETE: '[data-id=drawer-delete]',
  APPROVE: '[data-id=approve]',
  DENY: '[data-id=deny]',

  SOURCE_NODE_HEADER: '[data-id=source-header]',
  SOURCE_NODE: '[data-id=source-1]',
  DESTINATION_NODE: '[data-id=destination-0]',
  ACTION_NODE: '[data-id=action-0]',
  INSTRUMENTATION_RULE_NODE: '[data-id=rule-0]',

  ACTION_DROPDOWN_OPTION: '[data-id=option-pii-masking]',
  MULTI_SOURCE_CONTROL: '[data-id=multi-source-control]',

  TITLE: '[data-id=title]',
  SOURCE_TITLE: '[data-id=sourceName]',
  CHECKBOX: '[data-id=checkbox]',

  NOTIF_MANAGER_BUTTON: '[data-id=notif-manager-button]',
  NOTIF_MANAGER_CONTENR: '[data-id=notif-manager-content]',
};

export const BUTTONS = {
  NEXT: 'NEXT',
  DONE: 'DONE',
  ADD_DESTINATION: 'ADD DESTINATION',
  UNINSTRUMENT: 'Uninstrument',
};

export const INPUTS = {
  ACTION_DROPDOWN: 'Type to search...',
};

const CYPRESS_TEST = 'Cypress Test';

export const TEXTS = {
  UPDATED_NAME: CYPRESS_TEST,

  NO_RESOURCES: (namespace: string) => `No resources found in ${namespace} namespace.`,

  SOURCE_WARN_MODAL_TITLE: 'Uninstrument 5 sources',
  SOURCE_WARN_MODAL_NOTE: "You're about to uninstrument the last source",
  DESTINATION_WARN_MODAL_TITLE: `Delete destination (${CYPRESS_TEST})`,
  DESTINATION_WARN_MODAL_NOTE: "You're about to delete the last destination",
  ACTION_WARN_MODAL_TITLE: `Delete action (${CYPRESS_TEST})`,
  INSTRUMENTATION_RULE_WARN_MODAL_TITLE: `Delete rule (${CYPRESS_TEST})`,

  NOTIF_SOURCES_CREATED: 'Successfully created 5 sources',
  NOTIF_SOURCES_DELETED: 'Successfully deleted 5 sources',
};
