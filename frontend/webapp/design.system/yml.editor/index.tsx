import React, { useState } from 'react';
import YamlEditor from '@focus-reactive/react-yaml';
import styled from 'styled-components';
import theme from '@/styles/palette';
import { Copied, Copy } from '@/assets/icons/app';

const Container = styled.div`
  position: relative;
  background-color: ${theme.colors.blue_grey};
  border-radius: 8px;
  padding: 4px;

  div {
    color: #f5b175;
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
    color: #f5b175;
  }
  .cm-gutters {
    display: none;
    border-top-left-radius: 8px;
    border-top-right-radius: 8px;
  }
`;

const EditorOverlay = styled.div`
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 10; // Ensure this is higher than the editor's z-index
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
    setData(data);
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
    <>
      <Container>
        <CopyIconWrapper onClick={handleCopy}>
          {isCopied ? (
            <Copied style={{ width: 18, height: 18 }} />
          ) : (
            <Copy style={{ width: 18, height: 18 }} />
          )}
        </CopyIconWrapper>

        <div style={{ position: 'relative' }}>
          <YamlEditor
            key={JSON.stringify(data)}
            json={data}
            onChange={handleChange}
          />
          <EditorOverlay />
        </div>
      </Container>
    </>
  );
};
