import React, { useId } from 'react';
import styled from 'styled-components';
import { DataTab, Divider, InstrumentStatus, MonitorsIcons, Status, Text, Tooltip } from '@/reuseable-components';
import { capitalizeFirstLetter, getProgrammingLanguageIcon, parseJsonStringToPrettyString, safeJsonParse, WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';

export enum DataCardFieldTypes {
  DIVIDER = 'divider',
  MONITORS = 'monitors',
  ACTIVE_STATUS = 'active-status',
  SOURCE_CONTAINER = 'source-container',
}

export interface DataCardRow {
  type?: DataCardFieldTypes;
  title?: string;
  tooltip?: string;
  value?: string;
  width?: string;
}

interface Props {
  data: DataCardRow[];
}

const ListContainer = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 16px 32px;
  width: 100%;
`;

const ListItem = styled.div<{ $width: string }>`
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: ${({ $width }) => $width};
`;

const ItemTitle = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  font-size: 10px;
  line-height: 16px;
`;

export const DataCardFields: React.FC<Props> = ({ data }) => {
  return (
    <ListContainer>
      {data.map(({ type, title, tooltip, value, width = 'unset' }) => {
        const id = useId();

        return (
          <ListItem key={id} $width={width}>
            <Tooltip text={tooltip} withIcon>
              {!!title && <ItemTitle>{title}</ItemTitle>}
            </Tooltip>
            {renderValue(type, value)}
          </ListItem>
        );
      })}
    </ListContainer>
  );
};

const PreWrap = styled(Text)`
  font-size: 12px;
  white-space: pre-wrap;
`;

const renderValue = (type: DataCardRow['type'], value: DataCardRow['value']) => {
  // We need to maintain this with new components every time we add a new type to "DataCardFieldTypes"

  switch (type) {
    case DataCardFieldTypes.DIVIDER:
      return <Divider length='100%' margin='0' />;

    case DataCardFieldTypes.MONITORS:
      return <MonitorsIcons monitors={value?.split(', ') || []} withTooltips size={14} />;

    case DataCardFieldTypes.ACTIVE_STATUS:
      return <Status isActive={value == 'true'} withIcon withBorder withSmaller withSpecialFont />;

    case DataCardFieldTypes.SOURCE_CONTAINER: {
      const { containerName, language, runtimeVersion } = safeJsonParse(value, {
        containerName: '-',
        language: WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN,
        runtimeVersion: '-',
      });

      return (
        <DataTab title={containerName} subTitle={`${capitalizeFirstLetter(language)} • Runtime: ${runtimeVersion}`} logo={getProgrammingLanguageIcon(language)}>
          <InstrumentStatus language={language} />
        </DataTab>
      );
    }

    default: {
      return <PreWrap>{parseJsonStringToPrettyString(value || '-')}</PreWrap>;
    }
  }
};
