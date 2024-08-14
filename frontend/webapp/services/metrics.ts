import { API } from '@/utils/constants';
import { get } from './api';

export async function getOverviewMetrics() {
  return await get(API.OVERVIEW_METRICS);
}
