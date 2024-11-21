import React, { PropsWithChildren, useCallback, useEffect, useState } from 'react';
import styled from 'styled-components';
import type { IStyledComponentBase, Keyframes, Substitute } from 'styled-components/dist/types';

interface HookProps {
  container: IStyledComponentBase<'web', Substitute<React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement>, {}>> & string;
  animateIn: Keyframes;
  animateOut: Keyframes;
  duration?: number; // in milliseconds
}

type TransitionProps = PropsWithChildren<{ enter: boolean }>;

export const useTransition = ({ container, animateIn, animateOut, duration = 300 }: HookProps) => {
  const Animated = styled(container)<{ $isEntering: boolean; $isLeaving: boolean }>`
    animation-name: ${({ $isEntering, $isLeaving }) => ($isEntering ? animateIn : $isLeaving ? animateOut : 'none')};
    animation-duration: ${duration}ms;
    animation-fill-mode: forwards;
  `;

  const Transition = useCallback(({ children, enter }: TransitionProps) => {
    const [mounted, setMounted] = useState(false);

    useEffect(() => {
      const t = setTimeout(() => setMounted(enter), duration + 50); // +50ms to ensure the animation is finished
      return () => clearTimeout(t);
    }, [enter, duration]);

    return (
      <Animated $isEntering={enter} $isLeaving={!enter && mounted}>
        {children}
      </Animated>
    );

    // do not add dependencies here, it will cause re-renders which we want to avoid
  }, []);

  return Transition;
};
