import React from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import styled from 'styled-components';
import type { SourceContainer } from '@/types';
import { Badge, Text } from '@/reuseable-components';
import { getProgrammingLanguageIcon, getStatusIcon } from '@/utils';

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

const Row = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 12px;
  background-color: ${({ theme }) => theme.colors.white_opacity['004']};
`;

const LanguageIcon = styled.div<{ $isError?: boolean }>`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: linear-gradient(180deg, rgba(249, 249, 249, 0.06) 0%, rgba(249, 249, 249, 0.02) 100%);
`;

const RowBody = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
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
          // TODO: replace with actual status (only after we changed the "get sources" to include uninstrumented & instrumented sources)
          const active = true;

          return (
            <Row key={`container-${idx}`}>
              <LanguageIcon>
                <Image src={getProgrammingLanguageIcon(language)} width={20} height={20} alt='source' />
              </LanguageIcon>

              <RowBody>
                <Text size={14}> {containerName}</Text>
                <Text size={10} color={theme.text.grey} style={{ textTransform: 'capitalize' }}>
                  {language}
                  <span style={{ textTransform: 'none' }}> • Runtime: {runtimeVersion}</span>
                </Text>
              </RowBody>

              <InstrumentStatus $active={active}>
                <Image src={active ? getStatusIcon('success') : '/icons/common/circled-cross.svg'} alt='' width={12} height={12} />
                <Text size={10} family='secondary' color={active ? theme.text.success : theme.text.grey}>
                  {active ? 'Instrumented' : 'Uninstrumented'}
                </Text>
              </InstrumentStatus>
            </Row>
          );
        })}
      </Body>
    </Container>
  );
};
