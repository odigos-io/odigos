import Logs from "@/assets/icons/logs-grey.svg";
import LogsFocus from "@/assets/icons/logs-blue.svg";
import Metrics from "@/assets/icons/chart-line-grey.svg";
import MetricsFocus from "@/assets/icons/chart-line-blue.svg";
import Traces from "@/assets/icons/tree-structure-grey.svg";
import TracesFocus from "@/assets/icons/tree-structure-blue.svg";

export const MONITORING_OPTIONS = [
  {
    id: 1,
    icons: {
      notFocus: () => Logs(),
      focus: () => LogsFocus(),
    },
    title: "Logs",
    tapped: true,
  },
  {
    id: 2,
    icons: {
      notFocus: () => Metrics(),
      focus: () => MetricsFocus(),
    },
    title: "Metrics",
    tapped: true,
  },
  {
    id: 3,
    icons: {
      notFocus: () => Traces(),
      focus: () => TracesFocus(),
    },
    title: "Traces",
    tapped: true,
  },
];
