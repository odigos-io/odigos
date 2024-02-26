import React from 'react';
import { Link } from '@keyval-dev/design-system';

interface KeyvalLinkProps {
  value: string;
  onClick?: () => void;
  fontSize?: number;
}

export function KeyvalLink(props: KeyvalLinkProps) {
  return <Link {...props} />;
}
