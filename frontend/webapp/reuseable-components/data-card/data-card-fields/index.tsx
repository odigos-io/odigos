import React, { useId } from 'react';
import styled from 'styled-components';
import { NOTIFICATION_TYPE } from '@/types';
import { ActiveStatus, Code, DataTab, Divider, InstrumentStatus, MonitorsIcons, NotificationNote, Text, Tooltip } from '@/reuseable-components';
import { capitalizeFirstLetter, getProgrammingLanguageIcon, parseJsonStringToPrettyString, safeJsonParse, WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';

export enum DataCardFieldTypes {
  DIVIDER = 'divider',
  MONITORS = 'monitors',
  ACTIVE_STATUS = 'active-status',
  SOURCE_CONTAINER = 'source-container',
  CODE = 'code',
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
      return <MonitorsIcons monitors={value?.split(', ') || []} withLabels />;

    case DataCardFieldTypes.ACTIVE_STATUS:
      return <ActiveStatus isActive={value == 'true'} size={10} withIcon withBorder />;

    case DataCardFieldTypes.SOURCE_CONTAINER: {
      const { containerName, language, runtimeVersion, otherAgent, hasPresenceOfOtherAgent } = safeJsonParse(value, {
        containerName: '-',
        language: WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN,
        runtimeVersion: '-',
        otherAgent: null,
        hasPresenceOfOtherAgent: false,
      });

      // Determine if running concurrently is possible based on language and other_agent
      const canRunInParallel = !hasPresenceOfOtherAgent && (language === WORKLOAD_PROGRAMMING_LANGUAGES.PYTHON || language === WORKLOAD_PROGRAMMING_LANGUAGES.JAVA);

      return (
        <DataTab
          title={containerName}
          subTitle={`${language === WORKLOAD_PROGRAMMING_LANGUAGES.JAVASCRIPT ? 'Node.js' : capitalizeFirstLetter(language)} • Runtime Version: ${runtimeVersion}`}
          logo={getProgrammingLanguageIcon(language)}
          isExtended={!!otherAgent}
          renderExtended={() => (
            <NotificationNote
              type={NOTIFICATION_TYPE.INFO}
              message={
                hasPresenceOfOtherAgent
                  ? `By default, we do not operate alongside the ${otherAgent}. Please contact the Odigos team for guidance on enabling this configuration.`
                  : canRunInParallel
                  ? `We are operating alongside the ${otherAgent}, which is not the recommended configuration. We suggest disabling the ${otherAgent} for optimal performance.`
                  : `Concurrent execution with the ${otherAgent} is not supported. Please disable one of the agents to enable proper instrumentation.`
              }
            />
          )}
          renderActions={() => <InstrumentStatus language={language} />}
        />
      );
    }

    case DataCardFieldTypes.CODE: {
      const params = safeJsonParse(value, { language: '', code: '' });

      return <Code {...params} />;
    }

    default: {
      return <PreWrap>{parseJsonStringToPrettyString(value || '-')}</PreWrap>;
    }
  }
};
