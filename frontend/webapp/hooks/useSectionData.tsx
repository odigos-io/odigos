import { useMemo, useState } from 'react';

export function useSectionData(initialData) {
  const [sectionData, setSectionData] = useState(initialData);

  const totalSelected = useMemo(() => {
    let total = 0;
    for (const key in sectionData) {
      const apps = sectionData[key]?.objects;
      const counter = apps?.filter(
        (item) => item.selected && !item.app_instrumentation_labeled
      )?.length;
      total += counter;
    }

    return total;
  }, [JSON.stringify(sectionData)]);

  return { sectionData, setSectionData, totalSelected };
}
