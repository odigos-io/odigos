import React, { useEffect, useMemo, useState } from 'react';
import { useQuery } from 'react-query';
import { KeyvalButton, KeyvalLoader, KeyvalText } from '@/design.system';
import { OVERVIEW, QUERIES, ROUTES } from '@/utils';
import { getApplication, getNamespaces } from '@/services';
import {
  LoaderWrapper,
  SectionContainerWrapper,
} from './sources.section.styled';
import { NamespaceAccordion } from './namespace-accordion';
import styled from 'styled-components';
import theme from '@/styles/palette';
import { useSources } from '@/hooks';
import { useRouter } from 'next/navigation';
import { setSources } from '@/store';
import { useDispatch } from 'react-redux';

const TitleWrapper = styled.div`
  margin-bottom: 24px;
`;

const ButtonWrapper = styled.div`
  position: absolute;
  bottom: 24px;
  width: 90%;
`;

interface AccordionItem {
  name: string;
  kind: string;
  instances: number;
  app_instrumentation_labeled: boolean;
  ns_instrumentation_labeled: boolean;
  instrumentation_effective: boolean;
  selected: boolean;
}

interface AccordionData {
  title: string;
  items: AccordionItem[];
}

export function FastSourcesSelection({ sectionData, setSectionData }) {
  const [accordionData, setAccordionData] = useState<AccordionData[]>([]);
  const router = useRouter();
  const dispatch = useDispatch();
  const { isLoading, data: namespaces } = useQuery(
    [QUERIES.API_NAMESPACES],
    getNamespaces
  );

  const { upsertSources } = useSources();

  useEffect(() => {
    if (namespaces) {
      const accordionData = namespaces.namespaces.map((item, index) => ({
        title: item.name,
        items:
          sectionData?.[item.name]?.objects.map((app) => ({
            ...app,
            selected: app.selected,
          })) || [],
      }));
      setAccordionData(accordionData);
    }
  }, [namespaces]);

  const handleSetCurrentNamespace = async (selectedNs) => {
    const currentNsApps = await getApplication(selectedNs.title);

    const updatedSectionData = {
      ...sectionData,
      [selectedNs.title]: {
        selected_all: sectionData[selectedNs.title]?.selected_all || false,
        future_selected:
          sectionData[selectedNs.title]?.future_selected || false,
        objects: currentNsApps.applications.map((app) => ({
          ...app,
          selected: sectionData[selectedNs.title]?.objects?.find(
            (a) => a.name === app.name
          )?.selected,
        })),
      },
    };

    const accordionData = namespaces.namespaces.map((item) => ({
      title: item.name,
      items: updatedSectionData[item.name]?.objects
        .map((app) => ({
          ...app,
          selected: app.selected,
        }))
        .filter((app) => !app.instrumentation_effective),
    }));

    // Update both sectionData and accordionData
    setSectionData(updatedSectionData);
    setAccordionData(accordionData);
  };

  const onSelectItemChange = (item: AccordionItem, ns: string) => {
    const updatedAccordionData = accordionData.map((a_data: AccordionData) => {
      if (a_data.title === ns) {
        return {
          ...a_data,
          items: a_data.items.map((i: AccordionItem) => {
            if (i.name === item.name) {
              return { ...i, selected: !i.selected };
            }
            return i;
          }),
        };
      }
      return a_data;
    });

    const updatedSectionData = {
      ...sectionData,
      [ns]: {
        ...sectionData[ns],
        selected_all: accordionData[ns]
          ?.find((a) => a.title === ns)
          .items.every((i) => i.selected),
        future_selected: accordionData[ns]
          ?.find((a) => a.title === ns)
          .items.some((i) => !i.selected),
        objects: sectionData[ns].objects.map((a) => {
          if (a.name === item.name) {
            return { ...a, selected: !a.selected };
          }
          return a;
        }),
      },
    };

    setSectionData(updatedSectionData);

    // Update the accordion data state with the modified data
    setAccordionData(updatedAccordionData);
  };

  const onSelectAllChange = (ns, value) => {
    const updatedAccordionData = accordionData.map((a_data: AccordionData) => {
      if (a_data.title === ns) {
        return {
          ...a_data,
          items: a_data.items.map((i: AccordionItem) => {
            return { ...i, selected: value };
          }),
        };
      }
      return a_data;
    });

    const updatedSectionData = {
      ...sectionData,
      [ns]: {
        ...sectionData[ns],
        objects: sectionData[ns]?.objects?.map((a) => {
          return { ...a, selected: value };
        }),
      },
    };

    // Update the accordion data state with the modified data
    setSectionData(updatedSectionData);
    setAccordionData(updatedAccordionData);
  };

  function onConnectClick() {
    const isSetup = window.location.pathname.includes('choose-sources');

    if (isSetup) {
      dispatch(setSources(sectionData));
      router.push(ROUTES.CHOOSE_DESTINATION);
      return;
    }

    upsertSources({
      sectionData,
      onSuccess: () => router.push(`${ROUTES.SOURCES}?poll=true`),
      onError: null,
    });
  }

  if (isLoading) {
    return (
      <LoaderWrapper>
        <KeyvalLoader />
      </LoaderWrapper>
    );
  }
  const isSetup = window.location.pathname.includes('choose-sources');
  return (
    <SectionContainerWrapper>
      <TitleWrapper>
        <KeyvalText>Fast Sources Selection</KeyvalText>
      </TitleWrapper>
      <div style={{ height: '75vh' }}>
        <NamespaceAccordion
          data={accordionData}
          onSelectItem={onSelectItemChange}
          setCurrentNamespace={(data) => handleSetCurrentNamespace(data)}
          onSelectAllChange={onSelectAllChange}
        />
      </div>
      <ButtonWrapper>
        <KeyvalButton onClick={onConnectClick}>
          <KeyvalText weight={600} color={theme.text.dark_button}>
            {isSetup ? 'Next' : OVERVIEW.CONNECT}
          </KeyvalText>
        </KeyvalButton>
      </ButtonWrapper>
    </SectionContainerWrapper>
  );
}
