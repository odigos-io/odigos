import React, { PropsWithChildren, useEffect, useState } from 'react';
import styled from 'styled-components';
import type { IStyledComponentBase, Keyframes, Substitute } from 'styled-components/dist/types';

interface Props {
  container: IStyledComponentBase<'web', Substitute<React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement>, {}>> & string;
  animateIn: Keyframes;
  animateOut: Keyframes;
  enter: boolean;
}

const Animated = (Container: Props['container']) => styled(Container)<{
  $isEntering: boolean;
  $isLeaving: boolean;
  $animateIn: Props['animateIn'];
  $animateOut: Props['animateOut'];
}>`
  animation: ${({ $isEntering, $isLeaving, $animateIn, $animateOut }) => ($isEntering ? $animateIn : $isLeaving ? $animateOut : 'none')} 0.3s forwards;
`;

export const Transition: React.FC<PropsWithChildren<Props>> = ({ container: Container, children, animateIn, animateOut, enter }) => {
  const AnimatedContainer = Animated(Container);
  const [isEntered, setIsEntered] = useState(false);

  useEffect(() => {
    if (enter) setIsEntered(true);
  }, [enter]);

  if (!enter && !isEntered) return null;

  return (
    <AnimatedContainer $isEntering={enter} $isLeaving={isEntered && !enter} $animateIn={animateIn} $animateOut={animateOut}>
      {children}
    </AnimatedContainer>
  );
};
