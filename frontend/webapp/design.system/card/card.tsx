import React from 'react';
import { Card } from '@keyval-dev/design-system';

interface CardProps {
  children: JSX.Element | JSX.Element[];
  focus?: any;
  type?: string;
  header?: {
    title?: string;
    subtitle?: string;
    body?: () => JSX.Element | JSX.Element[];
  };
}
export function KeyvalCard(props: CardProps) {
  return <Card {...props}>{props.children}</Card>;
}
