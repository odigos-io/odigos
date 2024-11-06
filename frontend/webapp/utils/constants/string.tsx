export const SETUP = {
  STEPS: {
    CHOOSE_SOURCE: 'Choose Source',
    CHOOSE_DESTINATION: 'Choose Destination',
    CREATE_CONNECTION: 'Create Connection',
    STATUS: {
      ACTIVE: 'active',
      DISABLED: 'disabled',
      DONE: 'done',
    },
    ID: {
      CHOOSE_SOURCE: 'choose-source',
      CHOOSE_DESTINATION: 'choose-destination',
      CREATE_CONNECTION: 'create-connection',
    },
  },
  HEADER: {
    CHOOSE_SOURCE_TITLE: 'Select applications to connect',
    CHOOSE_DESTINATION_TITLE: 'Add new backend destination from the list',
  },
  MENU: {
    NAMESPACES: 'Namespaces',
    SELECT_ALL: 'Select All',
    FUTURE_APPLY: 'Apply for any future apps',
    TOOLTIP: 'Automatically connect any future apps in this namespace',
    SEARCH_PLACEHOLDER: 'Search',
    TYPE: 'Type',
    MONITORING: 'I want to monitor',
  },
  NEXT: 'Next',
  BACK: 'Back',
  ALL: 'All',
  CLEAR_SELECTION: 'Clear Selection',
  APPLICATIONS: 'Applications',
  RUNNING_INSTANCES: 'Running Instances',
  SELECTED: 'Selected',
  SOURCE_SELECTED: 'Source selected',
  NONE_SOURCE_SELECTED: 'No source selected',
  MANAGED: 'Managed',
  CREATE_CONNECTION: 'Create Connection',
  UPDATE_CONNECTION: 'Update Connection',
  CONNECTION_MONITORS: 'This connection will monitor:',
  MONITORS: {
    LOGS: 'Logs',
    METRICS: 'Metrics',
    TRACES: 'Traces',
  },
  DESTINATION_NAME: 'Destination Name',
  CREATE_DESTINATION: 'Create Destination',
  UPDATE_DESTINATION: 'Update Destination',
  QUICK_HELP: 'Quick Help',
  ERROR: 'Something went wrong',
};

export const INPUT_TYPES = {
  INPUT: 'input',
  DROPDOWN: 'dropdown',
  MULTI_INPUT: 'multiInput',
  KEY_VALUE_PAIR: 'keyValuePairs',
  TEXTAREA: 'textarea',
};

export const OVERVIEW = {
  ODIGOS: 'Odigos',
  MENU: {
    OVERVIEW: 'Overview',
    SOURCES: 'Sources',
    DESTINATIONS: 'Destinations',
    ACTIONS: 'Actions',
    INSTRUMENTATION_RULES: 'Instrumentation Rules',
  },
  SEARCH_SOURCE: 'Search Source',
  ADD_NEW_SOURCE: 'Add New Source',
  ADD_NEW_DESTINATION: 'Add New Destination',
  ADD_NEW_ACTION: 'Add New Action',
  EMPTY_DESTINATION: 'No destinations found',
  EMPTY_ACTION: 'No actions found',
  EMPTY_SOURCE: 'No sources found in this namespace',
  DESTINATION_UPDATE_SUCCESS: 'Destination updated successfully',
  DESTINATION_CREATED_SUCCESS: 'Destination created successfully',
  DESTINATION_DELETED_SUCCESS: 'Destination deleted successfully',
  SOURCE_UPDATE_SUCCESS: 'Source updated successfully',
  SOURCE_CREATED_SUCCESS: 'Source created successfully',
  SOURCE_DELETED_SUCCESS: 'Source deleted successfully',
  ACTION_UPDATE_SUCCESS: 'Action updated successfully',
  ACTION_UPDATE_ERROR: 'Failed to update action',
  MANAGE: 'Manage',
  DELETE: 'Delete',
  DELETE_DESTINATION: 'Delete Destination',
  DELETE_SOURCE: 'Delete Source',
  DELETE_ACTION: 'Delete Action',

  SOURCE_DANGER_ZONE_TITLE: 'Delete this source',
  ACTION_DANGER_ZONE_TITLE: 'Delete this action',
  SOURCE_DANGER_ZONE_SUBTITLE:
    'Uninstrument this source, and delete all odigos associated data. You can always re-instrument this source later with odigos.',
  ACTION_DANGER_ZONE_SUBTITLE: 'This action cannot be undone. This will permanently delete the action and all associated data.',
  DELETE_MODAL_TITLE: 'Delete this destination',
  DELETE_MODAL_SUBTITLE: 'This action cannot be undone. This will permanently delete the destination and all associated data.',
  DELETE_BUTTON: 'I want to delete this destination',
  CONFIRM_SOURCE_DELETE: 'I want to delete this source',
  CONFIRM_DELETE_ACTION: 'I want to delete this action',
  CONNECT: 'Connect',
  REPORTED_NAME: 'Override service.name',
  CREATE_ACTION: 'Create Action',
  EDIT_ACTION: 'Edit Action',
  ACTION_DESCRIPTION:
    'Actions are a way to modify the OpenTelemetry data recorded by Odigos Sources, before it is exported to your Odigos Destinations.',
  CREATE_INSTRUMENTATION_RULE: 'Create Instrumentation Rule',
  EDIT_INSTRUMENTATION_RULE: 'Edit Instrumentation Rule',
  INSTRUMENTATION_RULE_DESCRIPTION: 'Instrumentation Rules control how telemetry is recorded from your application.',
};

export const ACTION = {
  SAVE: 'Save',
  CONTACT_US: 'Contact Us',
  LEARN_MORE: 'Learn more',
  LINK_TO_DOCS: 'Link to docs',
  ENABLE: 'Enable',
  DISABLE: 'Disable',
  RUNNING: 'Running',
  APPLIED: 'Applied',
  DELETE_ALL: 'Delete All',
  CREATE: 'Create',
  UPDATE: 'Update',
  DELETE: 'Delete',
};

export const FORM_ALERTS = {
  REQUIRED_FIELDS: 'Required fields are missing!',
};

export const NOTIFICATION = {
  ERROR: 'error',
  SUCCESS: 'success',
};

export const PARAMS = {
  STATUS: 'status',
  DELETED: 'deleted',
  CREATED: 'created',
  UPDATED: 'updated',
};

//odigos actions
export const ACTIONS = {
  MONITORS_TITLE: 'Monitors',
  ACTION_NAME: 'Action Name',
  ACTION_NOTE: 'Note',
  NOTE_PLACEHOLDER: 'Add a note to describe the use case of this action',
  CREATE_ACTION: 'Create Action',
  UPDATE_ACTION: 'Update Action',

  AddClusterInfo: {
    TITLE: 'Add Cluster Info',
    DESCRIPTION: `The “Add Cluster Info” Odigos Action can be used to add resource attributes to telemetry signals originated from the k8s cluster where the Odigos is running.`,
  },
  DeleteAttribute: {
    TITLE: 'Delete Attribute',
    DESCRIPTION: `The “Delete Attribute” Odigos Action can be used to delete attributes from telemetry signals originated from the k8s cluster where the Odigos is running.`,
  },
  RenameAttribute: {
    TITLE: 'Rename Attribute',
    DESCRIPTION: `The “Rename Attribute” Odigos Action can be used to rename attributes from telemetry signals originated from the k8s cluster where the Odigos is running.`,
  },
  ErrorSampler: {
    TITLE: 'Error Sampler',
    DESCRIPTION: `The “Error Sampler” Odigos Action is a Global Action that supports error sampling by filtering out non-error traces.`,
  },
  ProbabilisticSampler: {
    TITLE: 'Probabilistic Sampler',
    DESCRIPTION: `The “Probabilistic Sampler” Odigos Action supports probabilistic sampling based on a configured sampling percentage applied to the TraceID.`,
  },
  LatencySampler: {
    TITLE: 'Latency Sampler',
    DESCRIPTION: `The “Latency Sampler” Odigos Action is an Endpoint Action that samples traces based on their duration for a specific service and endpoint (HTTP route) filter.`,
  },
  PiiMasking: {
    TITLE: 'PII Masking',
    DESCRIPTION: `The “PII Masking” Odigos Action is an Endpoint Action that masks PII (Personally Identifiable Information) attributes from telemetry signals.`,
  },
  SEARCH_ACTION: 'Search Action',
};

export const INSTRUMENTATION_RULES = {
  'payload-collection': {
    TITLE: 'Payload Collection',
    DESCRIPTION: 'Collect span attributes containing payload data to traces.',
  },
};

export const MONITORS = {
  LOGS: 'Logs',
  METRICS: 'Metrics',
  TRACES: 'Traces',
};
