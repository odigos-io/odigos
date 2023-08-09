import { SETUP } from "@/utils/constants";
import {
  Metrics,
  MetricsFocus,
  Traces,
  TracesFocus,
  LogsFocus,
  Logs,
} from "@/assets/icons/monitors";

export type MonitoringOption = {
  title: string;
  tapped: boolean;
  icons: object;
  id: number;
};

export const MONITORING_OPTIONS = [
  {
    id: 1,
    icons: {
      notFocus: () => <Logs />,
      focus: () => <LogsFocus />,
    },
    title: SETUP.MONITORS.LOGS,
    type: "logs",
    tapped: true,
  },
  {
    id: 2,
    icons: {
      notFocus: () => <Metrics />,
      focus: () => <MetricsFocus />,
    },
    title: SETUP.MONITORS.METRICS,
    type: "metrics",
    tapped: true,
  },
  {
    id: 3,
    icons: {
      notFocus: () => <Traces />,
      focus: () => <TracesFocus />,
    },
    title: SETUP.MONITORS.TRACES,
    type: "traces",
    tapped: true,
  },
];
