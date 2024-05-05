import React from 'react';
import { Link } from '@odigos-io/design-system';

interface KeyvalLinkProps {
  value: string;
  onClick?: () => void;
  fontSize?: number;
}

export function KeyvalLink(props: KeyvalLinkProps) {
  return <Link {...props} />;
}
