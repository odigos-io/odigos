import { Collector } from "@/types/collectors";
import { ApplicationData } from "@/types/apps";
import { OverviewDestResponseItem } from "@/types/dests";

export interface OverviewApiResponse {
  collectors: Collector[];
  dests: OverviewDestResponseItem[];
  sources: ApplicationData[];
}
