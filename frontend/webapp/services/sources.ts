import { API } from '@/utils/constants';
import { SelectedSources } from '@/types/sources';
import { get, post, httpDelete, patch } from './api';

export async function getNamespaces() {
  return await get(API.NAMESPACES);
}

export async function getApplication(id: string) {
  return await get(`${API.APPLICATIONS}/${id}`);
}

export async function setNamespaces(body: SelectedSources): Promise<void> {
  return await post(API.NAMESPACES, body);
}

export async function getSources() {
  return await get(API.SOURCES);
}

export async function getSource(namespace: string, kind: string, name: string) {
  return await get(
    `${API.SOURCES}/namespace/${namespace}/kind/${kind}/name/${name}`
  );
}

export async function deleteSource(
  namespace: string,
  kind: string,
  name: string
) {
  return await httpDelete(
    `${API.SOURCES}/namespace/${namespace}/kind/${kind}/name/${name}`
  );
}

export async function patchSources(
  namespace: string,
  kind: string,
  name: string,
  body: any
) {
  patch(
    `${API.SOURCES}/namespace/${namespace}/kind/${kind}/name/${name}`,
    body
  );
}
