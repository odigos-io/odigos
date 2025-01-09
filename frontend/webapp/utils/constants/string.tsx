export const SETUP = {
  MONITORS: {
    LOGS: 'Logs',
    METRICS: 'Metrics',
    TRACES: 'Traces',
  },
};

export const INPUT_TYPES = {
  INPUT: 'input',
  DROPDOWN: 'dropdown',
  MULTI_INPUT: 'multiInput',
  KEY_VALUE_PAIR: 'keyValuePairs',
  TEXTAREA: 'textarea',
  CHECKBOX: 'checkbox',
};

export enum CRUD {
  CREATE = 'Create',
  UPDATE = 'Update',
  DELETE = 'Delete',
}

export const ACTION = {
  SAVE: 'Save',
  CONTACT_US: 'Contact Us',
  LEARN_MORE: 'Learn more',
  LINK_TO_DOCS: 'Link to docs',
  ENABLE: 'Enable',
  DISABLE: 'Disable',
  RUNNING: 'Running',
  APPLIED: 'Applied',
  FETCH: 'Fetch',
  CREATE: CRUD.CREATE,
  UPDATE: CRUD.UPDATE,
  DELETE: CRUD.DELETE,
  DELETE_ALL: 'Delete All',
};

export const FORM_ALERTS = {
  REQUIRED_FIELDS: 'Required fields are missing',
  FIELD_IS_REQUIRED: 'This field is required',
  FORBIDDEN: 'Forbidden',
  CANNOT_EDIT_RULE: 'Cannot edit instrumentation rule of this type',
  LATENCY_HTTP_ROUTE: 'HTTP route must start with a forward slash "/"',
};

export const BACKEND_BOOLEAN = {
  FALSE: 'False',
  TRUE: 'True',
};

export const INSTUMENTATION_STATUS = {
  INSTRUMENTED: 'Instrumented',
  UNINSTRUMENTED: 'Uninstrumented',
};

export const DATA_CARDS = {
  ACTION_DETAILS: 'Action Details',
  RULE_DETAILS: 'Instrumentation Rule Details',
  DESTINATION_DETAILS: 'Destination Details',
  SOURCE_DETAILS: 'Source Details',
  DETECTED_CONTAINERS: 'Detected Containers',
  DETECTED_CONTAINERS_DESCRIPTION: 'The system automatically instruments the containers it detects with a supported programming language.',
  DESCRIBE_SOURCE: 'Describe Source',
  DESCRIBE_ODIGOS: 'Describe Odigos',
  API_TOKENS: 'API Tokens',
};

export const DISPLAY_TITLES = {
  ACTION: 'Action',
  ACTIONS: 'Actions',
  INSTRUMENTATION_RULE: 'Instrumentation Rule',
  INSTRUMENTATION_RULES: 'Instrumentation Rules',
  DESTINATION: 'Destination',
  DESTINATIONS: 'Destinations',
  SOURCE: 'Source',
  SOURCES: 'Sources',

  NAMESPACE: 'Namespace',
  CONTAINER_NAME: 'Container Name',
  KIND: 'Kind',
  TYPE: 'Type',
  NAME: 'Name',
  NOTES: 'Notes',
  STATUS: 'Status',
  LANGUAGE: 'Language',
  MONITORS: 'Monitors',
  SIGNALS_FOR_PROCESSING: 'Signals for Processing',
};
