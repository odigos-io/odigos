import { useRef } from 'react';
import { DOCS_LINK } from '@/utils';
import styled from 'styled-components';
import { NotebookIcon } from '@/assets';
import { Button, ButtonProps } from '..';

const StyledButton = styled(Button)`
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  gap: 6px;
  min-width: 100px;
`;

export const DocsButton = ({ endpoint = '/', variant = 'secondary' }: { endpoint?: string; variant?: ButtonProps['variant'] }) => {
  const ref = useRef<HTMLButtonElement>(null);

  return (
    <StyledButton
      ref={ref}
      variant={variant}
      onClick={() => {
        window.open(`${DOCS_LINK}${endpoint}`, '_blank', 'noopener noreferrer');
        ref.current?.blur();
      }}
    >
      <NotebookIcon size={18} />
      Docs
    </StyledButton>
  );
};
