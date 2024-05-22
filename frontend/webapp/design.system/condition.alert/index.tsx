import React from 'react';
import styled from 'styled-components';
import theme from '@/styles/palette';
import { Condition } from '@/types';
import { KeyvalText } from '../text/text';
import { GreenCheck, RedError } from '@/assets/icons/app';

interface ConditionAlertProps {
  conditions: Condition[] | undefined;
  title?: string;
}

const ConditionAlertContainer = styled.div`
  border: 1px solid ${theme.colors.blue_grey};
  border-radius: 8px;
  padding: 10px;
  margin-top: 8px;
`;

const ConditionItem = styled.div`
  padding: 10px;
`;

const ConditionIconContainer = styled.div``;

const ConditionDetails = styled.div`
  display: flex;
  flex-wrap: wrap;
  margin-top: 4px;
`;

const ConditionSeparator = styled.div`
  width: 2px;
  height: 30px;
  border-radius: 20px;
  background-color: ${theme.colors.blue_grey};
  margin-left: 10px;
`;

const IconWrapper = styled.div<{ bgColor: string }>`
  width: 32px;
  height: 32px;
  border-radius: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: ${({ bgColor }) => bgColor};
`;

const InnerIconWrapper = styled.div<{ borderColor: string }>`
  width: 16px;
  height: 16px;
  border-radius: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid ${({ borderColor }) => borderColor};
`;

export const Conditions: React.FC<ConditionAlertProps> = ({
  conditions,
  title = 'Status',
}) => {
  const getSuccessIcon = () => (
    <IconWrapper bgColor="#3fb94f40">
      <InnerIconWrapper borderColor="#3fb950">
        <GreenCheck style={{ width: 10, height: 10 }} />
      </InnerIconWrapper>
    </IconWrapper>
  );

  const getErrorIcon = () => (
    <IconWrapper bgColor="#f8524952">
      <InnerIconWrapper borderColor="#f85249">
        <RedError
          style={{ width: 10, height: 10, marginLeft: 2, marginBottom: 2 }}
        />
      </InnerIconWrapper>
    </IconWrapper>
  );

  return conditions ? (
    <div>
      <KeyvalText size={14} weight={600}>
        {title}
      </KeyvalText>
      <ConditionAlertContainer>
        {conditions.map((condition, index) => (
          <ConditionItem key={index}>
            <div style={{ display: 'flex', gap: '8px' }}>
              <ConditionIconContainer>
                {condition.status === 'True'
                  ? getSuccessIcon()
                  : getErrorIcon()}
              </ConditionIconContainer>
              <div>
                <KeyvalText size={14}>{condition.message}</KeyvalText>
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
              </div>
            </div>
            {index !== conditions.length - 1 && <ConditionSeparator />}
          </ConditionItem>
        ))}
      </ConditionAlertContainer>
    </div>
  ) : null;
};
