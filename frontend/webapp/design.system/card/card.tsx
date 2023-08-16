import React from 'react';
import { Card } from '@keyval-dev/design-system';

interface CardProps {
  children: JSX.Element | JSX.Element[];
  focus?: any;
}
export function KeyvalCard(props: CardProps) {
  return <Card {...props}>{props.children}</Card>;
}
