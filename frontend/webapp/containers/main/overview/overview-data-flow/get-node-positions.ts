import { getValueForRange } from '@/utils';
import nodeConfig from './node-config.json';
import { Types } from '@odigos/ui-components';

const { nodeWidth, nodeHeight } = nodeConfig;

interface Params {
  containerWidth: number;
}

export type NodePositions = Record<
  Types.ENTITY_TYPES,
  {
    x: number;
    y: (idx?: number) => number;
  }
>;

export const getNodePositions = ({ containerWidth }: Params) => {
  const startX = 24;
  const endX = (containerWidth <= 1500 ? 1500 : containerWidth) - nodeWidth - startX;
  const getY = (idx?: number) => nodeHeight * ((idx || 0) + 1);

  const positions: NodePositions = {
    [Types.ENTITY_TYPES.INSTRUMENTATION_RULE]: {
      x: startX,
      y: getY,
    },
    [Types.ENTITY_TYPES.SOURCE]: {
      x: getValueForRange(containerWidth, [
        [0, 1600, endX / 3.5],
        [1600, null, endX / 4],
      ]),
      y: getY,
    },
    [Types.ENTITY_TYPES.ACTION]: {
      x: getValueForRange(containerWidth, [
        [0, 1600, endX / 1.55],
        [1600, null, endX / 1.6],
      ]),
      y: getY,
    },
    [Types.ENTITY_TYPES.DESTINATION]: {
      x: endX,
      y: getY,
    },
  };

  return positions;
};
