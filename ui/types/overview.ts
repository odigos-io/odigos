import { Collector } from "@/types/collectors";
import { ApplicationData } from "@/types/apps";
import { DestResponseItem } from "@/types/dests";

export interface OverviewApiResponse {
  collectors: Collector[];
  dests: DestResponseItem[];
  sources: ApplicationData[];
}
