import { API } from "@/utils/constants";
import { get, post, put } from "./api";

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

export async function updateDestination(body: any, id: string) {
  return await put(`${API.DESTINATIONS}/${id}`, body);
}
