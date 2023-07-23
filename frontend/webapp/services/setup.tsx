import { API } from "@/utils/constants";
import { get, post } from "./api";

export async function getNamespaces() {
  return await get(API.NAMESPACES);
}

export async function getApplication(id: string) {
  return await get(`${API.APPLICATIONS}/${id}`);
}

export async function setNamespaces(body: any) {
  return await post(API.NAMESPACES, body);
}

export async function getDestinationsTypes() {
  return await get(API.DESTINATION_TYPE);
}

export async function getDestinations() {
  return await get(API.DESTINATIONS);
}

export async function getDestination(type: string) {
  return await get(`${API.DESTINATION_TYPE}/${type}`);
}

export async function setDestination(body: any) {
  return await post(API.DESTINATIONS, body);
}
