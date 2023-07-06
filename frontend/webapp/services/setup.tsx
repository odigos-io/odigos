import axios from "axios";
import { API } from "@/utils/constants";

export async function getNamespaces() {
  const { data, status } = await axios.get(API.NAMESPACES);
  if (status === 200) {
    return data;
  }
}

export async function getApplication(id: string) {
  const { data, status } = await axios.get(`${API.APPLICATIONS}/${id}`);
  if (status === 200) {
    return data;
  }
}
