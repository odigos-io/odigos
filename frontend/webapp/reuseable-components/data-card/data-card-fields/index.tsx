import React, { Fragment, useId } from 'react';
import styled from 'styled-components';
import { DataTab, Divider, InstrumentStatus, MonitorsIcons, Status, Text, Tooltip } from '@/reuseable-components';
import { capitalizeFirstLetter, getProgrammingLanguageIcon, parseJsonStringToPrettyString, safeJsonParse, WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';

export interface DataCardRow {
  type?: 'divider' | 'break-line' | 'monitors' | 'active-status' | 'source-container';
  title?: string;
  tooltip?: string;
  value?: string;
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

const ListItem = styled.div<{ $isStretched: boolean }>`
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: ${({ $isStretched }) => ($isStretched ? '100%' : 'unset')};
`;

const ItemTitle = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  font-size: 10px;
  line-height: 16px;
`;

const ItemValue = styled(Text)`
  color: ${({ theme }) => theme.colors.text};
  font-size: 12px;
  line-height: 18px;
`;

const PreWrap = styled(Text)`
  font-size: 12px;
  white-space: pre-wrap;
`;

const STRETCH_TYPES = ['source-container']; // Types that should stretch to 100% width

export const DataCardFields: React.FC<Props> = ({ data }) => {
  return (
    <ListContainer>
      {data.map(({ type, title, tooltip, value }) => {
        const id = useId();

        return (
          <ListItem key={id} $isStretched={!!type && STRETCH_TYPES.includes(type)}>
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

const renderValue = (type: DataCardRow['type'], value: DataCardRow['value']) => {
  switch (type) {
    case 'divider': {
      return <Divider length='585px' margin='0 auto' />;
    }

    case 'monitors': {
      return <MonitorsIcons monitors={value?.split(', ') || []} withTooltips size={14} />;
    }

    case 'active-status': {
      return <Status isActive={value == 'true'} withIcon withBorder withSmaller withSpecialFont />;
    }

    case 'source-container': {
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
      const str = parseJsonStringToPrettyString(value || '-');

      return (
        <ItemValue>
          <PreWrap>{str}</PreWrap>
        </ItemValue>
      );
    }
  }
};
