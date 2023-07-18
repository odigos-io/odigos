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

export async function getDestinations() {
  return await get(API.DESTINATION);
}

export async function getDestination(type: string) {
  return await get(`${API.DESTINATION}/${type}`);
}

export async function setDestination(body: any) {
  console.log("object", body);
  return await post(API.DESTINATION, body);
}
