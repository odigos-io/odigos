import axios from "axios";

export async function get(url: string) {
  const { data, status } = await axios.get(url);
  if (status === 200) {
    return data;
  }
}

export function post(url: string, body: any) {
  axios.post(url, body);
}

export function put(url: string, body: any) {
  axios.put(url, body);
}

export function httpDelete(url: string) {
  axios.delete(url);
}

export function patch(url: string, body: any) {
  axios.patch(url, body);
}

export async function patch(url: string, body: any) {
  await axios.patch(url, body);
}
