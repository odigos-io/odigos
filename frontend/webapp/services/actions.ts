import { API } from '@/utils';
import { get, post } from './api';
import { ActionItem, ActionData } from '@/types';

export async function setAction(body: ActionItem): Promise<void> {
  return post(API.SET_ACTION('AddClusterInfo'), body);
}

export async function getActions(): Promise<ActionData[]> {
  return get(API.ACTIONS);
}
