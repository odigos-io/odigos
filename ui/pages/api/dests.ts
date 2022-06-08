import type { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";

async function CreateNewDestination(
  req: NextApiRequest,
  res: NextApiResponse<any>
) {
  console.log(JSON.stringify(req.body));
  res.redirect("/");
}

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<any>
) {
  await CreateNewDestination(req, res);
}
