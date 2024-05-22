import React from 'react';
import { GreenCheck, RedError } from '@/assets/icons/app';
import styled from 'styled-components';
import { KeyvalText } from '@/design.system';

const Container = styled.div`
  display: inline-block;
  position: relative;
`;

const StatusIcon = styled.div`
  font-size: 24px;
  cursor: pointer;
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
  const healthyCount = conditions.filter(
    (condition) => condition.status === 'True'
  ).length;
  const totalCount = conditions.length;
  const allHealthy = healthyCount === totalCount;

  return (
    <Container>
      <StatusIcon>{allHealthy ? <GreenCheck /> : <RedError />}</StatusIcon>
      <Tooltip>
        <KeyvalText
          size={12}
        >{`${healthyCount}/${totalCount} checks OK`}</KeyvalText>
      </Tooltip>
    </Container>
  );
};
