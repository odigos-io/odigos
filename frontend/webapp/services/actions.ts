import { post } from './api';
import { API } from '@/utils';
import { ActionItem } from '@/types';

export async function setAction(body: ActionItem): Promise<void> {
  return post(API.SET_ACTION('AddClusterInfo'), body);
}
