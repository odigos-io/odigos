import axios from "axios";

export async function get(url: string) {
  const { data, status } = await axios.get(url);
  if (status === 200) {
    return data;
  }
}

export async function post(url: string, body: any) {
  const { data, status } = await axios.post(url, body);

  if (status === 200) {
    return data;
  }
}

export async function put(url: string, body: any) {
  const { data, status } = await axios.put(url, body);

  if (status === 200) {
    return data;
  }
}

export async function httpDelete(url: string) {
  const { data, status } = await axios.delete(url);

  if (status === 200) {
    return data;
  }
}
