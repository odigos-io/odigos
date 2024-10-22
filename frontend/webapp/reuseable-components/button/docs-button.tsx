import Image from 'next/image';
import styled from 'styled-components';
import { Button, ButtonProps } from '.';
import { DOCS_LINK } from '@/utils';

const StyledButton = styled(Button)`
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  gap: 6px;
  min-width: 100px;
`;

export const DocsButton = ({ endpoint = '/', variant = 'secondary' }: { endpoint?: string; variant?: ButtonProps['variant'] }) => {
  return (
    <StyledButton variant={variant} onClick={() => window.open(`${DOCS_LINK}${endpoint}`, '_blank', 'noopener noreferrer')}>
      <Image src='/icons/common/notebook.svg' alt='Docs' width={18} height={18} />
      Docs
    </StyledButton>
  );
};
