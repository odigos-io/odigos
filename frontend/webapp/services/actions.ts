import { API } from '@/utils';
import { get, httpDelete, post, put } from './api';
import { ActionItem, ActionData } from '@/types';

export async function setAction(body: ActionItem, type: string): Promise<void> {
  return post(API.SET_ACTION(type), body);
}

export async function putAction(
  id: string = '',
  body: ActionItem,
  type: string
): Promise<void> {
  return put(API.PUT_ACTION(type, id), body);
}

export async function deleteAction(
  id: string,
  type: string = 'AddClusterInfo'
): Promise<void> {
  return httpDelete(API.DELETE_ACTION(type, id));
}

export async function getActions(): Promise<ActionData[]> {
  return get(API.ACTIONS);
}
