import { useState } from 'react';

export const useCopy = () => {
  const [isCopied, setIsCopied] = useState(false);
  const [copiedIndex, setCopiedIndex] = useState(-1);

  const clickCopy = (str: string, idx?: number) => {
    if (!isCopied) {
      setIsCopied(true);
      if (idx !== undefined) setCopiedIndex(idx);

      navigator.clipboard.writeText(str);

      setTimeout(() => {
        setIsCopied(false);
        setCopiedIndex(-1);
      }, 1000);
    }
  };

  return { isCopied, copiedIndex, clickCopy };
};
