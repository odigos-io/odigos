import React, { PropsWithChildren, useEffect, useState } from 'react';
import styled from 'styled-components';
import type { IStyledComponentBase, Keyframes, Substitute } from 'styled-components/dist/types';

interface Props {
  container: IStyledComponentBase<'web', Substitute<React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>, {}>> & string;
  animateIn: Keyframes;
  animateOut: Keyframes;
  enter: boolean;
}

export const Transition: React.FC<PropsWithChildren<Props>> = ({ container: Container, children, animateIn, animateOut, enter }) => {
  const [isEntered, setIsEntered] = useState(false);

  useEffect(() => {
    if (enter) setIsEntered(true);
  }, [enter]);

  const AnimatedContainer = styled(Container)<{ $isEntering: boolean; $isLeaving: boolean }>`
    animation: ${({ $isEntering, $isLeaving }) => ($isEntering ? animateIn : $isLeaving ? animateOut : 'none')} 0.3s forwards;
  `;

  return (
    <AnimatedContainer $isEntering={enter} $isLeaving={isEntered && !enter}>
      {children}
    </AnimatedContainer>
  );
};
