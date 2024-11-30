// DetectedContainers.tsx
import { KeyvalText } from '@/design.system';
import theme from '@/styles/palette';
import { Condition } from '@/types';
import React from 'react';
import styled from 'styled-components';

// Define the types for the language object
interface Language {
  container_name: string;
  language: string;
  runtime_version?: string;
  other_agent?: { [name: string]: string };
}

interface DetectedContainersProps {
  languages: Language[];
  conditions: Condition[];
}

const Container = styled.div`
  margin-top: 16px;
  max-width: 36vw;
  margin-bottom: 24px;
  border: 1px solid #374a5b;
  border-radius: 8px;
  padding: 24px;
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
  conditions,
}) => {
  const hasError = conditions.some((condition) => condition.status === 'False');
  return (
    <Container>
      <KeyvalText size={18} weight={600}>
        Detected Containers:
      </KeyvalText>
      <List>
        {languages.map((lang) => {
          const isInstrumented =
            lang.language !== 'ignore' &&
            lang.language !== 'unknown' &&
            !lang?.other_agent;
          return (
            <ListItem key={lang.container_name}>
              <KeyvalText
                color={!isInstrumented ? '#4caf50' : theme.text.light_grey}
              >
                {lang.container_name} (Language: {lang.language}
                {lang?.runtime_version
                  ? `, Runtime: ${lang.runtime_version}`
                  : ''}
                ){isInstrumented && !hasError && ' - Instrumented'}
              </KeyvalText>
              {lang.other_agent && lang.other_agent.name && (
                <KeyvalText
                  color={theme.colors.orange_brown}
                  size={12}
                  style={{ marginTop: 6 }}
                >
                  {`We are running concurrently with the ${lang.other_agent.name}, which is not recommended. It is advisable to disable other agents for optimal performance and compatibility.`}
                </KeyvalText>
              )}
            </ListItem>
          );
        })}
      </List>
      <KeyvalText size={14} color={theme.text.light_grey}>
        Note: The system automatically instruments the containers it detects
        with a supported programming language.
      </KeyvalText>
    </Container>
  );
};

export { DetectedContainers };
