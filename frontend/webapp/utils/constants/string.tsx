import type { NotificationType } from '@/types';

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

export const NOTIFICATION: {
  [key: string]: NotificationType;
} = {
  ERROR: 'error',
  SUCCESS: 'success',
  WARNING: 'warning',
  INFO: 'info',
  DEFAULT: 'default',
};
