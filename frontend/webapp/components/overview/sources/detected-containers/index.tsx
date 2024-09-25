// DetectedContainers.tsx
import { KeyvalText } from '@/design.system';
import theme from '@/styles/palette';
import React from 'react';
import styled from 'styled-components';

// Define the types for the language object
interface Language {
  container_name: string;
  language: string;
  runtime_version?: string;
}

interface DetectedContainersProps {
  languages: Language[];
}

const Container = styled.div`
  margin-top: 16px;
  max-width: 36vw;
  margin-bottom: 24px;
`;

const List = styled.ul`
  list-style: disc;
`;

const ListItem = styled.li`
  padding: 2px 0;
  &::marker {
    color: ${theme.colors.white};
  }
`;

const DetectedContainers: React.FC<DetectedContainersProps> = ({
  languages,
}) => {
  // Find the first language that is not ignored

  return (
    <Container>
      <KeyvalText size={18} weight={600}>
        Detected Containers:
      </KeyvalText>
      <List>
        {languages.map((lang) => (
          <ListItem key={lang.container_name}>
            <KeyvalText
              color={
                lang.language !== 'ignore' && lang.language !== 'unknown'
                  ? '#4caf50'
                  : theme.text.light_grey
              }
            >
              {lang.container_name} (Language: {lang.language}
              {lang?.runtime_version
                ? `, Runtime: ${lang.runtime_version}`
                : ''}
              )
              <b>
                {lang.language !== 'ignore' &&
                  lang.language !== 'unknown' &&
                  ' - Instrumented'}
              </b>
            </KeyvalText>
          </ListItem>
        ))}
      </List>
      <KeyvalText size={14} color={theme.text.light_grey}>
        Note: The system automatically instruments the containers it detects
        with a supported programming language.
      </KeyvalText>
    </Container>
  );
};

export { DetectedContainers };
