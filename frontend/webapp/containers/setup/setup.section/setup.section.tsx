import React, { useEffect, useMemo, useState } from "react";
import {
  SetupContentWrapper,
  SetupSectionContainer,
} from "./setup.section.styled";
import { SetupHeader } from "../setup.header/setup.header";
import { SourcesList, SourcesOptionMenu } from "@/components/setup";
import { getApplication, setNamespaces } from "@/services/setup";

const DEFAULT_CONFIG = {
  selected_all: false,
  future_selected: false,
};

export function SetupSection({ namespaces }: any) {
  const [currentNamespace, setCurrentNamespace] = useState<any>(null);
  const [selectedApplications, setSelectedApplications] = useState<any>({});
  const [searchFilter, setSearchFilter] = useState<string>("");

  const namespacesList = useMemo(() => {
    return namespaces.map((item: any, index: number) => {
      return { id: index, label: item.name };
    });
  }, [namespaces]);

  useEffect(() => {
    !currentNamespace && setCurrentNamespace(namespaces[0]);
  }, [namespaces]);

  useEffect(() => {
    onNameSpaceChange();
  }, [currentNamespace]);

  const totalSelected = useMemo(() => {
    let total = 0;
    for (const key in selectedApplications) {
      const apps = selectedApplications[key]?.objects;
      const counter = apps?.filter((item: any) => item.selected)?.length;
      total += counter;
    }
    return total;
  }, [JSON.stringify(selectedApplications)]);

  const sourceData = useMemo(() => {
    const data = selectedApplications[currentNamespace?.name];
    return searchFilter
      ? data?.objects.filter((item: any) =>
          item.name.toLowerCase().includes(searchFilter.toLowerCase())
        )
      : data?.objects;
  }, [searchFilter, currentNamespace, selectedApplications]);

  async function onNameSpaceChange() {
    if (!currentNamespace || selectedApplications[currentNamespace?.name])
      return;
    const data = await getApplication(currentNamespace?.name);
    const newSelectedNamespace = {
      ...selectedApplications,
      [currentNamespace?.name]: {
        ...DEFAULT_CONFIG,
        objects: [...data?.applications],
      },
    };

    setSelectedApplications(newSelectedNamespace);
  }

  function handleSourceClick({ item }: any) {
    const objIndex = selectedApplications[
      currentNamespace?.name
    ].objects.findIndex((app) => app.name === item.name);

    const namespace = selectedApplications[currentNamespace?.name];
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
      };
    }
    handleSetNewSelectedConfig(currentNamespaceConfig);
  }

  function onSelectAllChange(value: boolean) {
    const namespace = selectedApplications[currentNamespace?.name];
    let objects = [...namespace.objects];
    objects.forEach((item) => {
      item.selected = value;
    });

    const currentNamespaceConfig = {
      ...namespace,
      selected_all: value,
      objects,
    };
    handleSetNewSelectedConfig(currentNamespaceConfig);
  }

  function onFutureApplyChange(value: boolean) {
    const currentNamespaceConfig = {
      ...selectedApplications[currentNamespace?.name],
      future_selected: value,
    };
    handleSetNewSelectedConfig(currentNamespaceConfig);
  }

  function handleSetNewSelectedConfig(config: any) {
    const newSelectedNamespaceConfig = {
      ...selectedApplications,
      [currentNamespace?.name]: config,
    };
    setSelectedApplications(newSelectedNamespaceConfig);
  }

  function onNextClick() {
    setNamespaces(selectedApplications);
  }

  return (
    <SetupSectionContainer>
      <SetupHeader onNextClick={onNextClick} totalSelected={totalSelected} />
      <SetupContentWrapper>
        <SourcesOptionMenu
          currentNamespace={currentNamespace}
          setCurrentItem={setCurrentNamespace}
          data={namespacesList}
          searchFilter={searchFilter}
          setSearchFilter={setSearchFilter}
          onSelectAllChange={onSelectAllChange}
          selectedApplications={selectedApplications}
          onFutureApplyChange={onFutureApplyChange}
        />

        <SourcesList
          data={sourceData}
          selectedData={selectedApplications[currentNamespace?.name]}
          onItemClick={handleSourceClick}
          onClearClick={() => onSelectAllChange(false)}
        />
      </SetupContentWrapper>
    </SetupSectionContainer>
  );
}
