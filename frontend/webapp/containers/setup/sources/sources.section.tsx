import React, { useEffect, useMemo, useState } from "react";
import { SourcesList, SourcesOptionMenu } from "@/components/setup";
import { getApplication, getNamespaces } from "@/services/setup";
import { useQuery } from "react-query";
import { QUERIES } from "@/utils/constants";

const DEFAULT_CONFIG = {
  selected_all: false,
  future_selected: false,
};

export function SourcesSection({ sectionData, setSectionData }: any) {
  const [currentNamespace, setCurrentNamespace] = useState<any>(null);
  const [searchFilter, setSearchFilter] = useState<string>("");

  const { isLoading, data } = useQuery([QUERIES.API_NAMESPACES], getNamespaces);

  const namespacesList = useMemo(() => {
    return data?.namespaces?.map((item: any, index: number) => {
      return { id: index, label: item.name };
    });
  }, [data]);

  useEffect(() => {
    !currentNamespace && setCurrentNamespace(data?.namespaces[0]);
  }, [data]);

  useEffect(() => {
    onNameSpaceChange();
  }, [currentNamespace]);

  const sourceData = useMemo(() => {
    const namespace = sectionData[currentNamespace?.name];
    return searchFilter
      ? namespace?.objects.filter((item: any) =>
          item.name.toLowerCase().includes(searchFilter.toLowerCase())
        )
      : namespace?.objects;
  }, [searchFilter, currentNamespace, sectionData]);

  async function onNameSpaceChange() {
    if (!currentNamespace || sectionData[currentNamespace?.name]) return;
    const namespace = await getApplication(currentNamespace?.name);
    const newSelectedNamespace = {
      ...sectionData,
      [currentNamespace?.name]: {
        ...DEFAULT_CONFIG,
        objects: [...namespace?.applications],
      },
    };

    setSectionData(newSelectedNamespace);
  }

  function handleSourceClick({ item }: any) {
    const objIndex = sectionData[currentNamespace?.name].objects.findIndex(
      (app) => app.name === item.name
    );

    const namespace = sectionData[currentNamespace?.name];
    let objects = [...namespace.objects];

    objects[objIndex].selected = !objects[objIndex].selected;

    let currentNamespaceConfig = {
      ...namespace,
      objects,
    };

    if (!objects[objIndex].selected && namespace.selected_all) {
      currentNamespaceConfig = {
        ...currentNamespaceConfig,
        selected_all: false,
        future_selected: false,
      };
    }
    handleSetNewSelectedConfig(currentNamespaceConfig);
  }

  function onSelectAllChange(value: boolean) {
    const namespace = sectionData[currentNamespace?.name];
    let objects = [...namespace.objects];
    objects.forEach((item) => {
      item.selected = value;
    });

    const currentNamespaceConfig = {
      future_selected: value,
      selected_all: value,
      objects,
    };
    handleSetNewSelectedConfig(currentNamespaceConfig);
  }

  function onFutureApplyChange(value: boolean) {
    const currentNamespaceConfig = {
      ...sectionData[currentNamespace?.name],
      future_selected: value,
    };
    handleSetNewSelectedConfig(currentNamespaceConfig);
  }

  function handleSetNewSelectedConfig(config: any) {
    const newSelectedNamespaceConfig = {
      ...sectionData,
      [currentNamespace?.name]: config,
    };
    setSectionData(newSelectedNamespaceConfig);
  }

  if (isLoading) {
    return null;
  }

  return (
    <>
      <SourcesOptionMenu
        currentNamespace={currentNamespace}
        setCurrentItem={setCurrentNamespace}
        data={namespacesList}
        searchFilter={searchFilter}
        setSearchFilter={setSearchFilter}
        onSelectAllChange={onSelectAllChange}
        selectedApplications={sectionData}
        onFutureApplyChange={onFutureApplyChange}
      />

      <SourcesList
        data={sourceData}
        selectedData={sectionData[currentNamespace?.name]}
        onItemClick={handleSourceClick}
        onClearClick={() => onSelectAllChange(false)}
      />
    </>
  );
}
