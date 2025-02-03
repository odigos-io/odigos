import type { ExportedSignals } from '@/types';
import { SIGNAL_TYPE } from '@odigos/ui-components';

export const extractMonitors = (exportedSignals: ExportedSignals) => {
  const filtered = Object.keys(exportedSignals).filter((signal) => exportedSignals[signal as SIGNAL_TYPE] === true) as SIGNAL_TYPE[];

  return filtered;
};
