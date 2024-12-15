import styled from 'styled-components';
import { Highlight, themes as prismThemes } from 'prism-react-renderer';
import { flattenObjectKeys, safeJsonParse, safeJsonStringify } from '@/utils';

interface Props {
  language: string;
  code: string;
  flatten?: boolean;
}

const Token = styled.span`
  white-space: pre-wrap;
  overflow-wrap: break-word;
  opacity: 0.75;
  font-size: 12px;
  font-family: ${({ theme }) => theme.font_family.code};
`;

export const Code: React.FC<Props> = ({ language, code, flatten }) => {
  const str = flatten && language === 'json' ? safeJsonStringify(flattenObjectKeys(safeJsonParse(code, {}))) : code;

  return (
    <Highlight theme={prismThemes.palenight} language={language} code={str}>
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
