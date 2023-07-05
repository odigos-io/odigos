import { API } from "@/utils/constants/urls";

export async function getConfig() {
  const response = await fetch(API.CONFIG);
  const data = await response.json();
  return data;
}
