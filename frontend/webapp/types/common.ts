export interface Condition {
  type: string;
  status: string;
  message: string;
  lastTransitionTime: string;
}

export interface Notification {
  id: string;
  message: string;
  title?: string;
  seen: boolean;
  isNew?: boolean;
  time?: string;
  target?: string;
  event?: string;
  crdType?: string;
  type: 'success' | 'error' | 'info';
}

export type Config = {
  config: {
    installation: string;
  };
};

export interface DropdownOption {
  id: string;
  value: string;
}

export interface StepProps {
  title: string;
  subtitle?: string;
  state: 'finish' | 'active' | 'disabled';
  stepNumber: number;
}

export enum OVERVIEW_ENTITY_TYPES {
  SOURCE = 'source',
  ACTION = 'action',
  DESTINATION = 'destination',
}

export enum OVERVIEW_NODE_TYPES {
  ADD_SOURCE = 'addSource',
  ADD_ACTION = 'addAction',
  ADD_DESTIONATION = 'addDestination',
}

export enum STATUSES {
  HEALTHY = 'healthy',
}
