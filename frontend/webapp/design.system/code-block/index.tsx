import React from 'react';
import { Code } from '@odigos-io/design-system';

interface CodeProps {
  text: string;
  title?: string;
  highlightedWord?: {
    primary: {
      words: string[];
      color: string;
    };
    secondary?: {
      words: string[];
      color: string;
    };
  };
  onCopy?: () => void;
}

export function CodeBlock(props: CodeProps) {
  return <Code {...props} />;
}
