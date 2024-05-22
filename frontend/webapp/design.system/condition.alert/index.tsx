import React from 'react';
import styled from 'styled-components';
import theme from '@/styles/palette';
import { Condition } from '@/types';
import { KeyvalText } from '../text/text';
import { Error, Success } from '@/assets/icons/app';

interface ConditionAlertProps {
  conditions: Condition[] | undefined;
}

const ConditionAlertContainer = styled.div`
  border: 1px solid ${theme.colors.blue_grey};
  border-radius: 5px;
  padding: 10px;
  margin-top: 8px;
`;

const ConditionItem = styled.div`
  padding: 10px;
`;

const ConditionIconContainer = styled.div`
  width: 20px;
  height: 20px;
`;

const ConditionDetails = styled.div`
  display: flex;
  flex-wrap: wrap;
  padding-left: 28px;
  margin-top: 4px;
`;

const ConditionSeparator = styled.div`
  width: 2px;
  height: 30px;
  border-radius: 20px;
  background-color: ${theme.colors.blue_grey};
  margin-left: 10px;
`;

export const Conditions: React.FC<ConditionAlertProps> = ({ conditions }) => {
  return conditions ? (
    <div>
      <KeyvalText weight={600}>Status</KeyvalText>
      <ConditionAlertContainer>
        {conditions.map((condition, index) => (
          <ConditionItem key={index}>
            <div style={{ display: 'flex', gap: '8px' }}>
              <ConditionIconContainer>
                {condition.status === 'True' ? <Success /> : <Error />}
              </ConditionIconContainer>
              <KeyvalText size={14}>{condition.message}</KeyvalText>
            </div>
            <ConditionDetails>
              <KeyvalText
                style={{ marginRight: '8px' }}
                color={theme.text.grey}
                size={12}
              >
                {condition.type}
              </KeyvalText>
              <KeyvalText color={theme.text.grey} size={12}>
                {condition.last_transition_time}
              </KeyvalText>
            </ConditionDetails>
            {index !== conditions.length - 1 && <ConditionSeparator />}
          </ConditionItem>
        ))}
      </ConditionAlertContainer>
    </div>
  ) : null;
};
