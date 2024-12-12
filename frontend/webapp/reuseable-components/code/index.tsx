import styled from 'styled-components';
import { Highlight, themes as prismThemes } from 'prism-react-renderer';

interface Props {
  language: string;
  code: string;
}

const Token = styled.span`
  white-space: pre-wrap;
  opacity: 0.75;
`;

export const Code: React.FC<Props> = ({ language, code }) => {
  return (
    <Highlight theme={prismThemes.palenight} language={language} code={code}>
      {({ getLineProps, getTokenProps, tokens }) => (
        <pre>
          {tokens.map((line, i) => (
            <div key={`line-${i}`} {...getLineProps({ line })}>
              {/* <span>{i + 1}</span> */}
              {line.map((token, ii) => (
                <Token key={`line-${i}-token-${ii}`} {...getTokenProps({ token })} />
              ))}
            </div>
          ))}
        </pre>
      )}
    </Highlight>
  );
};
