import axios from "axios";
import { API } from "@/utils/constants/urls";

export async function getConfig() {
  const { data, status } = await axios.get(API.CONFIG);
  if (status === 200) {
    return data;
  }
}
