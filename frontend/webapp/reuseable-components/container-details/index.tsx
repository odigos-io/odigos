import React from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import styled from 'styled-components';
import type { SourceContainer } from '@/types';
import { Badge, DataTab, Text } from '@/reuseable-components';
import { capitalizeFirstLetter, getProgrammingLanguageIcon, getStatusIcon, INSTUMENTATION_STATUS, WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';

interface Props {
  containers: SourceContainer[];
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px 24px 24px 24px;
  border-radius: 24px;
  border: 1px solid ${({ theme }) => theme.colors.border};
`;

const Header = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

const Title = styled(Text)`
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
`;

const Description = styled(Text)`
  font-size: 12px;
  color: ${({ theme }) => theme.text.grey};
`;

const Body = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

const InstrumentStatus = styled.div<{ $active: boolean }>`
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 2px 6px;
  margin-left: auto;
  border-radius: 360px;
  border: 1px solid ${({ $active, theme }) => ($active ? theme.colors.dark_green : theme.colors.border)};
`;

export const ContainerDetails: React.FC<Props> = ({ containers }) => {
  return (
    <Container>
      <Header>
        <Title>
          Detected Containers <Badge label={containers.length} />
        </Title>
        <Description>The system automatically instruments the containers it detects with a supported programming language.</Description>
      </Header>

      <Body>
        {containers.map(({ containerName, language, runtimeVersion }, idx) => {
          const active = ![
            WORKLOAD_PROGRAMMING_LANGUAGES.IGNORED,
            WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN,
            WORKLOAD_PROGRAMMING_LANGUAGES.PROCESSING,
            WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS,
            WORKLOAD_PROGRAMMING_LANGUAGES.NO_RUNNING_PODS,
          ].includes(language);

          return (
            <DataTab key={`container-${idx}`} title={containerName} subTitle={`${capitalizeFirstLetter(language)} • Runtime: ${runtimeVersion}`} logo={getProgrammingLanguageIcon(language)}>
              <InstrumentStatus $active={active}>
                <Image src={active ? getStatusIcon('success') : '/icons/common/circled-cross.svg'} alt='' width={12} height={12} />
                <Text size={10} family='secondary' color={active ? theme.text.success : theme.text.grey}>
                  {active ? INSTUMENTATION_STATUS.INSTRUMENTED : INSTUMENTATION_STATUS.UNINSTRUMENTED}
                </Text>
              </InstrumentStatus>
            </DataTab>
          );
        })}
      </Body>
    </Container>
  );
};
