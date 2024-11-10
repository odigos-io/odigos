import React from 'react';
import styled from 'styled-components';
import { EdgeLabelRenderer, BaseEdge, type EdgeProps, type Edge, getSmoothStepPath } from '@xyflow/react';

const Label = styled.div<{ labelX: number; labelY: number; isError?: boolean }>`
  position: absolute;
  transform: ${({ labelX, labelY }) => `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`};
  width: 75px;
  padding: 2px 6px;
  background-color: ${({ theme }) => theme.colors.primary};
  border-radius: 360px;
  border: 1px solid ${({ isError, theme }) => (isError ? theme.colors.dark_red : theme.colors.border)};
  color: ${({ isError, theme }) => (isError ? theme.text.error : theme.text.light_grey)};
  font-family: ${({ theme }) => theme.font_family.secondary};
  font-size: 10px;
  font-weight: 400;
  text-transform: uppercase;
  display: flex;
  align-items: center;
  justify-content: center;
`;

const LabeledEdge: React.FC<EdgeProps<Edge<{ label: string; isMultiTarget?: boolean; isError?: boolean }>>> = ({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
  style,
}) => {
  const [edgePath] = getSmoothStepPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  return (
    <>
      <BaseEdge id={id} path={edgePath} style={style} />
      <EdgeLabelRenderer>
        <Label
          labelX={data?.isMultiTarget ? targetX - 50 : sourceX + 50}
          labelY={data?.isMultiTarget ? targetY : sourceY}
          isError={data?.isError}
          className='nodrag nopan'
        >
          {data?.label}
        </Label>
      </EdgeLabelRenderer>
    </>
  );
};

export default LabeledEdge;
