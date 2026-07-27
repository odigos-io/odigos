'use client';

import React, { type KeyboardEvent, type MouseEvent } from 'react';

export function templateRowActivationProps(onActivate: () => void): {
  role: 'button';
  tabIndex: 0;
  onClick: () => void;
  onKeyDown: (event: KeyboardEvent<HTMLElement>) => void;
} {
  return {
    role: 'button',
    tabIndex: 0,
    onClick: onActivate,
    onKeyDown: (event: KeyboardEvent<HTMLElement>) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        onActivate();
      }
    },
  };
}

export function stopRowActivation(event: { stopPropagation: () => void }): void {
  event.stopPropagation();
}
