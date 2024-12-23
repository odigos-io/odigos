import { useMemo } from 'react';
import styled from 'styled-components';
import { Highlight, themes as prismThemes } from 'prism-react-renderer';
import { flattenObjectKeys, removeEmptyValuesFromObject, safeJsonParse, safeJsonStringify } from '@/utils';

interface Props {
  language: string;
  code: string;
  flatten?: boolean;
}

const Token = styled.span`
  white-space: pre-wrap;
  overflow-wrap: break-word;
  font-size: 12px;
  font-family: ${({ theme }) => theme.font_family.code};
`;

export const Code: React.FC<Props> = ({ language, code, flatten }) => {
  const str = useMemo(() => {
    if (language === 'json') {
      const obj = safeJsonParse(code, {});
      const objNoNull = removeEmptyValuesFromObject(obj);

      if (flatten) return safeJsonStringify(flattenObjectKeys(objNoNull));
      return safeJsonStringify(objNoNull);
    }

    return code;
  }, [code, language, flatten]);

  return (
    <Highlight theme={prismThemes.palenight} language={language} code={str}>
      {({ getLineProps, getTokenProps, tokens }) => (
        <pre>
          {tokens.map((line, i) => (
            <div key={`line-${i}`} {...getLineProps({ line })}>
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
