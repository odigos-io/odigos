import React, { CSSProperties, DetailedHTMLProps, HTMLAttributes } from 'react';
import { useTheme } from 'styled-components';
import { createAnimation } from './helpers/animation';
import { cssValue, parseLengthAndUnit } from './helpers/unitConverter';

export type LengthType = number | string;

interface CommonProps extends DetailedHTMLProps<HTMLAttributes<HTMLSpanElement>, HTMLSpanElement> {
  color?: string;
  loading?: boolean;
  cssOverride?: CSSProperties;
  speedMultiplier?: number;
}

interface LoaderHeightWidthRadiusProps extends CommonProps {
  height?: LengthType;
  width?: LengthType;
  radius?: LengthType;
  margin?: LengthType;
}

const fade = createAnimation('FadeLoader', '50% {opacity: 0.3} 100% {opacity: 1}', 'fade');

export const FadeLoader = ({
  loading = true,
  color: clr,
  speedMultiplier = 1,
  cssOverride = {},
  height = 4,
  width = 1.5,
  radius = 2,
  margin = 2,
  ...additionalprops
}: LoaderHeightWidthRadiusProps) => {
  const theme = useTheme();
  const color = clr || theme.colors.text;

  const { value } = parseLengthAndUnit(margin);
  const radiusValue = value + 4.2;
  const quarter = radiusValue / 2 + radiusValue / 5.5;

  const wrapper: React.CSSProperties = {
    display: 'inherit',
    position: 'relative',
    fontSize: '0',
    top: radiusValue,
    left: radiusValue,
    width: `${radiusValue * 3}px`,
    height: `${radiusValue * 3}px`,
    ...cssOverride,
  };

  const style = (i: number): React.CSSProperties => {
    return {
      position: 'absolute',
      width: cssValue(width),
      height: cssValue(height),
      margin: cssValue(margin),
      backgroundColor: color,
      borderRadius: cssValue(radius),
      transition: '2s',
      animationFillMode: 'both',
      animation: `${fade} ${1.2 / speedMultiplier}s ${i * 0.12}s infinite ease-in-out`,
    };
  };

  const a: React.CSSProperties = {
    ...style(1),
    top: `${radiusValue}px`,
    left: '0',
  };
  const b: React.CSSProperties = {
    ...style(2),
    top: `${quarter}px`,
    left: `${quarter}px`,
    transform: 'rotate(-45deg)',
  };
  const c: React.CSSProperties = {
    ...style(3),
    top: '0',
    left: `${radiusValue}px`,
    transform: 'rotate(90deg)',
  };
  const d: React.CSSProperties = {
    ...style(4),
    top: `${-1 * quarter}px`,
    left: `${quarter}px`,
    transform: 'rotate(45deg)',
  };
  const e: React.CSSProperties = {
    ...style(5),
    top: `${-1 * radiusValue}px`,
    left: '0',
  };
  const f: React.CSSProperties = {
    ...style(6),
    top: `${-1 * quarter}px`,
    left: `${-1 * quarter}px`,
    transform: 'rotate(-45deg)',
  };
  const g: React.CSSProperties = {
    ...style(7),
    top: '0',
    left: `${-1 * radiusValue}px`,
    transform: 'rotate(90deg)',
  };
  const h: React.CSSProperties = {
    ...style(8),
    top: `${quarter}px`,
    left: `${-1 * quarter}px`,
    transform: 'rotate(45deg)',
  };

  if (!loading) {
    return null;
  }

  return (
    <span style={wrapper} {...additionalprops}>
      <span style={a} />
      <span style={b} />
      <span style={c} />
      <span style={d} />
      <span style={e} />
      <span style={f} />
      <span style={g} />
      <span style={h} />
    </span>
  );
};
