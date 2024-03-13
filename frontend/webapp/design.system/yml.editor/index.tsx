import React, { useState } from 'react';
import YamlEditor from '@focus-reactive/react-yaml';
import styled from 'styled-components';
import theme from '@/styles/palette';
import { Copied, Copy } from '@/assets/icons/app';

const Container = styled.div`
  position: relative;
  background-color: ${theme.colors.blue_grey};
  border-radius: 8px;
  width: fit-content;
  pointer-events: none;
  padding: 4px;
  div {
    color: ${theme.colors.light_grey};
  }
  .ͼb {
    color: #64a8fd;
  }
  .ͼm {
    color: ${theme.colors.white};
  }
  .ͼd {
    color: #f5b175;
  }
  .ͼc {
    color: ${theme.colors.white};
  }
  .cm-gutters {
    display: none;
    border-top-left-radius: 8px;
    border-top-right-radius: 8px;
  }
`;

const DisabledOverlay = styled.div`
  position: absolute;
  overflow-y: auto;
  max-height: 100%;
  border-radius: 8px;
  pointer-events: none;
  display: flex;
  align-items: center;
  justify-content: center;
`;

const CopyIconWrapper = styled.div`
  background-color: ${theme.colors.dark};
  z-index: 999;
  border-radius: 4px;
  padding: 4px;
  position: absolute;
  top: 5px;
  right: 5px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: all;
`;

export const YMLEditor = ({ data, setData }) => {
  const [isCopied, setIsCopied] = useState(false);

  const handleChange = (value) => {
    // setData(value);
  };

  const handleCopy = () => {
    navigator.clipboard
      .writeText(JSON.stringify(data, null, 2))
      .then(() => {
        setIsCopied(true);
        setTimeout(() => {
          setIsCopied(false);
        }, 3000);
      })
      .catch((err) => console.error('Error copying YAML to clipboard: ', err));
  };
  return (
    <DisabledOverlay>
      <Container>
        <CopyIconWrapper onClick={handleCopy}>
          {isCopied ? (
            <Copied style={{ width: 18, height: 18 }} />
          ) : (
            <Copy style={{ width: 18, height: 18 }} />
          )}
        </CopyIconWrapper>
        <YamlEditor
          key={JSON.stringify(data)}
          json={data}
          onChange={handleChange}
        />
      </Container>
    </DisabledOverlay>
  );
};
