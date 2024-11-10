import { get } from './api';
import { API } from '@/utils';

// Function to get Odigos description
export async function getOdigosDescription(): Promise<string> {
  return get(API.DESCRIBE_ODIGOS);
}

// Function to get source description based on namespace, kind, and name
export async function getSourceDescription(
  namespace: string,
  kind: string,
  name: string
): Promise<string> {
  return get(API.DESCRIBE_SOURCE(namespace, kind, name));
}
