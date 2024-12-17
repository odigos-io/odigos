import type { FC, MouseEventHandler } from 'react';

export * from './icons';

export type SVG = FC<{
  size?: number;
  fill?: string;
  rotate?: number;
  onClick?: MouseEventHandler<SVGSVGElement>;
}>;
