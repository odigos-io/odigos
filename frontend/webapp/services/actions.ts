import { API } from '@/utils';
import { get, post, put } from './api';
import { ActionItem, ActionData } from '@/types';

export async function setAction(body: ActionItem): Promise<void> {
  return post(API.SET_ACTION('AddClusterInfo'), body);
}

export async function putAction(
  id: string = '',
  body: ActionItem
): Promise<void> {
  return put(API.PUT_ACTION('AddClusterInfo', id), body);
}

export async function getActions(): Promise<ActionData[]> {
  return get(API.ACTIONS);
}
