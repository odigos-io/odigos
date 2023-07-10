import { API } from "@/utils/constants";
import { get } from "./api";

export async function getConfig() {
  return await get(API.CONFIG);
}
