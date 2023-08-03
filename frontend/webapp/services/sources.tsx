import { API } from "@/utils/constants";
import { get, post, httpDelete } from "./api";
import { SelectedSources } from "@/types/sources";

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

export async function deleteSource(
  namespace: string,
  kind: string,
  name: string
) {
  return await httpDelete(
    `${API.SOURCES}/namespace/${namespace}/kind/${kind}/name/${name}`
  );
}
