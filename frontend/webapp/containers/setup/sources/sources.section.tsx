import React, { useEffect, useMemo, useState } from 'react';
import { useQuery } from 'react-query';
import { useNotification } from '@/hooks';
import { KeyvalLoader } from '@/design.system';
import { NOTIFICATION, QUERIES } from '@/utils/constants';
import { getApplication, getNamespaces } from '@/services';
import { SourcesList, SourcesOptionMenu } from '@/components/setup';
import {
  LoaderWrapper,
  SectionContainerWrapper,
} from './sources.section.styled';

const DEFAULT = 'default';

export function SourcesSection({ sectionData, setSectionData }) {
  const [currentNamespace, setCurrentNamespace] = useState<any>(null);
  const [searchFilter, setSearchFilter] = useState<string>('');

  const { show, Notification } = useNotification();
  const { isLoading, data, isError, error } = useQuery(
    [QUERIES.API_NAMESPACES],
    getNamespaces
  );

  useEffect(() => {
    if (!currentNamespace && data) {
      const currentNamespace = data?.namespaces.find(
        (item: any) => item.name === DEFAULT
      );
      setCurrentNamespace(currentNamespace);
    }
  }, [data]);

  useEffect(() => {
    onNameSpaceChange();
  }, [currentNamespace]);

  useEffect(() => {
    isError &&
      show({
        type: NOTIFICATION.ERROR,
        message: error,
      });
  }, [isError]);

  const namespacesList = useMemo(
    () =>
      data?.namespaces?.map((item: any, index: number) => ({
        id: index,
        label: item.name,
      })),
    [data]
  );

  const sourceData = useMemo(() => {
    let namespace = sectionData[currentNamespace?.name];

    //filter by search query
    namespace = searchFilter
      ? namespace?.objects.filter((item: any) =>
          item.name.toLowerCase().includes(searchFilter.toLowerCase())
        )
      : namespace?.objects;
    //remove instrumented applications
    return namespace?.filter((item: any) => !item.instrumentation_effective);
  }, [searchFilter, currentNamespace, sectionData]);

  async function onNameSpaceChange() {
    if (!currentNamespace || sectionData[currentNamespace?.name]) return;
    const namespace = await getApplication(currentNamespace?.name);

    const { selected } = data.namespaces.find(
      (item) => item.name === currentNamespace?.name
    );

    const newSelectedNamespace = {
      ...sectionData,
      [currentNamespace?.name]: {
        selected_all: selected,
        future_selected: selected,
        objects: [...namespace?.applications],
      },
    };
    setSectionData(newSelectedNamespace);
    if (selected) {
      onSelectAllChange(true, newSelectedNamespace);
    }
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

  function onSelectAllChange(value: boolean, data?: any) {
    const currentData = data || sectionData;
    const namespace = currentData[currentNamespace?.name];
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
    return (
      <LoaderWrapper>
        <KeyvalLoader />
      </LoaderWrapper>
    );
  }

  return (
    <SectionContainerWrapper>
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
      <Notification />
    </SectionContainerWrapper>
  );
}
