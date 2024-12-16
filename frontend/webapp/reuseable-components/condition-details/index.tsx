import React, { useMemo, useState } from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { BACKEND_BOOLEAN, getStatusIcon } from '@/utils';
import { NOTIFICATION_TYPE, type Condition } from '@/types';
import { ExtendIcon, FadeLoader, Text } from '@/reuseable-components';

interface Props {
  conditions: Condition[];
}

const Container = styled.div<{ $hasErrors: boolean }>`
  border-radius: 24px;
  background-color: ${({ theme, $hasErrors }) => ($hasErrors ? theme.colors.darkest_red : theme.colors.white_opacity['004'])};
  cursor: pointer;
  &:hover {
    background-color: ${({ theme, $hasErrors }) => ($hasErrors ? theme.colors.error : theme.colors.white_opacity['008'])};
  }
  transition: background 0.3s;
`;

const Header = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 18px;
`;

const Body = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 6px 18px 12px 18px;
`;

const Row = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
`;

export const ConditionDetails: React.FC<Props> = ({ conditions }) => {
  const [extend, setExtend] = useState(false);

  const loading = useMemo(() => !conditions.length, [conditions]);
  const errors = useMemo(() => conditions.filter(({ status }) => status === BACKEND_BOOLEAN.FALSE), [conditions]);
  const hasErrors = !!errors.length;
  const headerText = loading ? 'Loading...' : hasErrors ? 'Operation Failed' : 'Operation Successful';

  return (
    <Container onClick={() => setExtend((prev) => !prev)} $hasErrors={hasErrors}>
      <Header>
        {loading ? <FadeLoader /> : <Image src={getStatusIcon(hasErrors ? NOTIFICATION_TYPE.ERROR : NOTIFICATION_TYPE.SUCCESS)} alt='' width={16} height={16} />}

        <Text color={hasErrors ? theme.text.error : theme.text.grey} size={14}>
          {headerText}
        </Text>
        <Text color={hasErrors ? theme.text.error_secondary : theme.text.dark_grey} size={12} family='secondary'>
          ({hasErrors ? errors.length : conditions.length}/{conditions.length})
        </Text>

        <ExtendIcon extend={extend} align='right' />
      </Header>

      {extend && (
        <Body>
          {conditions.map(({ status, message }, idx) => (
            <Row key={`condition-${idx}`}>
              <Image src={getStatusIcon(status === BACKEND_BOOLEAN.FALSE ? NOTIFICATION_TYPE.ERROR : NOTIFICATION_TYPE.SUCCESS)} alt='' width={14} height={14} />
              <Text color={status === BACKEND_BOOLEAN.FALSE ? theme.text.error : theme.text.darker_grey} size={12}>
                {message}
              </Text>
            </Row>
          ))}
        </Body>
      )}
    </Container>
  );
};
