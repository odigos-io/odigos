import { Types } from '@odigos/ui-components';
import type { ExportedSignals } from '@/types';

export const extractMonitors = (exportedSignals: ExportedSignals) => {
  const filtered = Object.keys(exportedSignals).filter((signal) => exportedSignals[signal as Types.SIGNAL_TYPE] === true) as Types.SIGNAL_TYPE[];

  return filtered;
};
