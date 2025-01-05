import React, { Fragment, useId, useMemo } from 'react';
import { Text } from '../text';
import { FlexRow } from '@/styles';
import theme from '@/styles/theme';
import { Tooltip } from '../tooltip';
import styled from 'styled-components';
import { Highlight, themes as prismThemes, type Token } from 'prism-react-renderer';
import { flattenObjectKeys, removeEmptyValuesFromObject, safeJsonParse, safeJsonStringify } from '@/utils';

interface Props {
  language: string;
  code: string;
  flatten?: boolean;
  pretty?: boolean;
}

const Table = styled.table`
  border-collapse: collapse;
  font-family: ${({ theme }) => theme.font_family.primary};
`;

const TableBody = styled.tbody``;

const TableRow = styled.tr``;

const TableData = styled.td`
  vertical-align: top;
  padding: 4px 6px;
`;

const CodeLineToken = styled.span<{ $isNoCode?: boolean }>`
  white-space: pre-wrap;
  overflow-wrap: break-word;
  font-size: 12px;
  font-family: ${({ theme, $isNoCode }) => ($isNoCode ? theme.font_family.primary : theme.font_family.code)};
`;

export const Code: React.FC<Props> = ({ language, code, flatten, pretty }) => {
  const str = useMemo(() => {
    if (language === 'json') {
      const obj = safeJsonParse(code, {});
      const objNoNull = removeEmptyValuesFromObject(obj);

      if (flatten) return safeJsonStringify(flattenObjectKeys(objNoNull));
      return safeJsonStringify(objNoNull);
    }

    return code;
  }, [code, language, flatten]);

  if (pretty && language === 'json') {
    return <PrettyJsonCode str={str} />;
  }

  return (
    <Highlight theme={prismThemes.palenight} language={language} code={str}>
      {({ getLineProps, getTokenProps, tokens }) => (
        <pre>
          {tokens.map((line, i) => (
            <div key={`line-${i}`} {...getLineProps({ line })}>
              {line.map((token, ii) => (
                <CodeLineToken key={`line-${i}-token-${ii}`} {...getTokenProps({ token })} />
              ))}
            </div>
          ))}
        </pre>
      )}
    </Highlight>
  );
};

const formatLineForPrettyMode = (line: Token[]) => {
  const ignoreTypes = ['punctuation', 'plain', 'operator'];

  return line
    .filter((token) => !ignoreTypes.includes(token.types[0]))
    .map(({ types, content }) => {
      const updatedTypes = [...types];
      const updatedContent = ['property', 'string'].includes(updatedTypes[0]) ? content.replace(/"/g, '') : content;

      // override types for prettier colors
      if (updatedTypes[0] === 'string') {
        if (['true', 'false'].includes(updatedContent.split('#')[0])) updatedTypes[0] = 'boolean';
        if (updatedContent.split('#')[0].match(/^[0-9]+$/)) updatedTypes[0] = 'number';
      }

      return {
        types: updatedTypes,
        content: updatedContent,
      };
    });
};

const getComponentsFromPropertyString = (propertyString: string) => {
  const [text, ...rest] = propertyString.split('#');
  const components =
    rest
      ?.map((c) => {
        if (!c.includes('=')) return null;
        const [type, value] = c.split('=');

        switch (type) {
          case 'tooltip':
            return <Tooltip key={propertyString} withIcon text={value} />;
          default:
            console.warn('unexpected component type!', type);
            return null;
        }
      })
      ?.filter((c) => !!c) || [];

  return { text, components };
};

const PrettyJsonCode: React.FC<{ str: string }> = ({ str }) => {
  const renderEmptyRows = (count: number = 2) => {
    const rows = new Array(count).fill((props) => (
      <TableRow {...props}>
        <TableData />
        <TableData />
      </TableRow>
    ));

    return (
      <Fragment>
        {rows.map((R, i) => (
          <R key={useId()} style={i === 0 ? { borderBottom: `1px solid ${theme.colors.border}` } : {}} />
        ))}
      </Fragment>
    );
  };

  return (
    <Highlight theme={prismThemes.vsDark} language='json' code={str}>
      {({ getLineProps, getTokenProps, tokens }) => (
        <Table>
          <TableBody>
            {tokens.map((line, i) => {
              const formattedLine = formatLineForPrettyMode(line);
              const lineProps = getLineProps({ line: formattedLine });

              if (formattedLine.length === 1 && formattedLine[0].types[0] === 'property') {
                return (
                  <Fragment key={`line-${i}`}>
                    {renderEmptyRows()}
                    <TableRow {...lineProps}>
                      <TableData>
                        <Text>{formattedLine[0].content}</Text>
                      </TableData>
                      <TableData />
                    </TableRow>
                  </Fragment>
                );
              } else if (formattedLine.length === 2) {
                return (
                  <TableRow key={`line-${i}`} {...lineProps}>
                    {formattedLine.map((token, ii) => {
                      const { children, ...tokenProps } = getTokenProps({ token });
                      const { text, components } = getComponentsFromPropertyString(children);

                      return (
                        <TableData key={`line-${i}-token-${ii}`}>
                          <FlexRow>
                            <CodeLineToken $isNoCode {...tokenProps}>
                              {text}
                            </CodeLineToken>
                            <FlexRow>{components}</FlexRow>
                          </FlexRow>
                        </TableData>
                      );
                    })}
                  </TableRow>
                );
              } else {
                if (!!formattedLine.length) console.warn('this line is unexpected!', i, formattedLine);
                return null;
              }
            })}
          </TableBody>
        </Table>
      )}
    </Highlight>
  );
};
