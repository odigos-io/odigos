import { QUERIES } from '@/utils/constants';
import { SelectedSources } from '@/types/sources';
import { useMutation, useQuery } from 'react-query';
import { getSources, setNamespaces } from '@/services';

export function useSources() {
  const { data: sources } = useQuery([QUERIES.API_SOURCES], getSources);

  const { mutateAsync } = useMutation((body: SelectedSources) =>
    setNamespaces(body)
  );

  async function upsertSources({ sectionData, onSuccess, onError }) {
    const sourceNamesSet = new Set(
      sources?.map((source: SelectedSources) => source.name)
    );
    const updatedSectionData: SelectedSources = {};

    for (const key in sectionData) {
      const { objects, ...rest } = sectionData[key];
      const updatedObjects = objects.map((item) => ({
        ...item,
        selected: item?.selected || sourceNamesSet.has(item.name),
      }));

      updatedSectionData[key] = {
        ...rest,
        objects: updatedObjects,
      };
    }

    try {
      await mutateAsync(updatedSectionData);
      if (onSuccess) {
        onSuccess();
      }
    } catch (error) {
      if (onError) {
        onError(error);
      }
    }
  }

  return { upsertSources };
}
