import React from 'react';
import YamlEditor from '@focus-reactive/react-yaml';
import styled from 'styled-components';
import theme from '@/styles/palette';

const Container = styled.div`
  background-color: ${theme.colors.blue_grey};
  border-radius: 8px;
  width: fit-content;
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
export const YMLEditor = ({ data, setData }) => {
  const handleChange = (value) => {
    console.log(value);
  };
  return (
    <Container>
      <YamlEditor json={data} onChange={handleChange} />
    </Container>
  );
};
