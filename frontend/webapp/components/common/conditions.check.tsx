import React from 'react';
import { GreenCheckIcon, RedErrorIcon } from '@keyval-dev/design-system';
import styled from 'styled-components';
import { KeyvalText } from '@/design.system';
import theme from '@/styles/palette';

const Container = styled.div`
  display: inline-block;
  position: relative;
`;

const StatusIcon = styled.div`
  font-size: 24px;
  cursor: pointer;
`;
const ProgressStatus = styled.div`
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: ${theme.colors.orange_brown};
`;

const Tooltip = styled.div`
  visibility: hidden;
  width: max-content;
  background-color: black;
  color: #fff;
  text-align: center;
  border-radius: 5px;
  padding: 8px;
  position: absolute;
  z-index: 1;
  bottom: 100%; /* Position above the icon */
  left: 50%;
  transform: translateX(-50%);
  opacity: 0;
  transition: opacity 0.3s;

  ${Container}:hover & {
    visibility: visible;
    opacity: 1;
  }
`;

export const ConditionCheck = ({ conditions }) => {
  const healthyCount = conditions?.filter(
    (condition) => condition.status === 'True'
  ).length;
  const totalCount = conditions?.length;
  const allHealthy = healthyCount === totalCount;

  return conditions ? (
    <Container>
      <StatusIcon>
        {allHealthy ? <GreenCheckIcon /> : <RedErrorIcon />}
      </StatusIcon>
      <Tooltip>
        <KeyvalText
          size={12}
        >{`${healthyCount}/${totalCount} checks OK`}</KeyvalText>
      </Tooltip>
    </Container>
  ) : (
    <Container>
      <ProgressStatus />
      <Tooltip>
        <KeyvalText size={12}>{'validating checks...'}</KeyvalText>
      </Tooltip>
    </Container>
  );
};
