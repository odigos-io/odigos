import axios from "axios";
import { API } from "@/utils/constants";

export async function getNamespaces() {
  const { data, status } = await axios.get(API.NAMESPACES);
  if (status === 200) {
    return data;
  }
}
