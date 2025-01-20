import type { ExportedSignals } from '@/types';

export const extractMonitors = (exportedSignals: ExportedSignals) => {
  const filtered = Object.keys(exportedSignals).filter((signal) => exportedSignals[signal as keyof ExportedSignals] === true);

  return filtered;
};
