import type { Action, ActionFormData } from '@odigos/ui-kit/types';

export type FetchedAction = Omit<ActionFormData, 'fields'> & {
  id: string;
  fields: Omit<ActionFormData['fields'], 'renames'> & {
    renames: string | null;
  };
};

export type ActionInput = Omit<FetchedAction, 'id'>;
