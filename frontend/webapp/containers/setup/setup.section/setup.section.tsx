import React, { useEffect, useMemo, useState } from "react";
import {
  EmptyListWrapper,
  SetupContentWrapper,
  SetupSectionContainer,
} from "./setup.section.styled";
import { SetupHeader } from "../setup.header/setup.header";
import { SourcesList, SourcesOptionMenu } from "@/components/setup";
import { useQuery } from "react-query";
import { getApplication, setNamespaces } from "@/services/setup";
import { QUERIES } from "@/utils/constants";
import Empty from "@/assets/images/empty-list.svg";

const DEFAULT_CONFIG = {
  selected_all: false,
  future_selected: false,
};

export function SetupSection({ namespaces }: any) {
  const [currentNamespace, setCurrentNamespace] = useState<any>(null);
  const [selectedApplications, setSelectedApplications] = useState<any>({});
  const [searchFilter, setSearchFilter] = useState<string>("");

  const namespacesList = useMemo(
    () =>
      namespaces.map((item: any, index: number) => {
        return { id: index, label: item.name };
      }),
    [namespaces]
  );

  const { data } = useQuery(
    [QUERIES.API_APPLICATIONS, currentNamespace],
    () => getApplication(currentNamespace.name),
    {
      // The query will not execute until the currentNamespace exists
      enabled: !!currentNamespace,
    }
  );

  useEffect(onNameSpaceChange, [data]);
  useEffect(() => {
    !currentNamespace && setCurrentNamespace(namespaces[0]);
  }, [namespaces]);

  function onNameSpaceChange() {
    if (!data || selectedApplications[currentNamespace?.name]) return;

    const newSelectedNamespace = {
      ...selectedApplications,
      [currentNamespace?.name]: {
        ...DEFAULT_CONFIG,
        objects: [...data?.applications],
      },
    };

    setSelectedApplications(newSelectedNamespace);
  }

  function getSourceData() {
    return searchFilter
      ? data?.applications.filter((item: any) =>
          item.name.toLowerCase().includes(searchFilter.toLowerCase())
        )
      : data?.applications;
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

    const newSelectedNamespaceConfig = {
      ...selectedApplications,
      [currentNamespace?.name]: currentNamespaceConfig,
    };
    setSelectedApplications(newSelectedNamespaceConfig);
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

    const newSelectedNamespaceConfig = {
      ...selectedApplications,
      [currentNamespace?.name]: currentNamespaceConfig,
    };
    setSelectedApplications(newSelectedNamespaceConfig);
  }

  function onFutureApplyChange(value: boolean) {
    const currentNamespaceConfig = {
      ...selectedApplications[currentNamespace?.name],
      future_selected: value,
    };

    const newSelectedNamespaceConfig = {
      ...selectedApplications,
      [currentNamespace?.name]: currentNamespaceConfig,
    };
    setSelectedApplications(newSelectedNamespaceConfig);
  }

  function onNextClick() {
    setNamespaces(selectedApplications);
    //[TODO]: Move To Next Step
  }

  return (
    <SetupSectionContainer>
      <SetupHeader onNextClick={onNextClick} />
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

        {!data?.applications?.length ? (
          <EmptyListWrapper>
            <Empty />
          </EmptyListWrapper>
        ) : (
          <SourcesList
            data={getSourceData()}
            selectedData={selectedApplications[currentNamespace?.name]}
            onItemClick={handleSourceClick}
            onClearClick={() => onSelectAllChange(false)}
          />
        )}
      </SetupContentWrapper>
    </SetupSectionContainer>
  );
}
