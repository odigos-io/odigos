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
  type: 'success' | 'error' | 'info';
}
