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
