type RulePathSegment = {
  wildcard: boolean;
  staticString: string;
  templateName: string;
};

export type TemplateSimulationResult =
  | { status: 'matched'; outputPath: string }
  | { status: 'no_match'; message: string }
  | { status: 'empty_input' };

function parseRuleTemplateName(raw: string): string {
  const name = raw.trim();
  return name || 'id';
}

export function parseUserInputRuleString(userInputRule: string): RulePathSegment[] {
  let segments = userInputRule.split('/');
  if (userInputRule.startsWith('/')) {
    segments = segments.slice(1);
  }

  return segments.map((segment) => {
    if (segment === '*') {
      return { wildcard: true, staticString: '', templateName: '' };
    }
    if (segment.startsWith('{') && segment.endsWith('}')) {
      return {
        wildcard: false,
        staticString: '',
        templateName: parseRuleTemplateName(segment.slice(1, -1)),
      };
    }
    return { wildcard: false, staticString: segment, templateName: '' };
  });
}

function attemptTemplateWithRule(pathSegments: string[], ruleSegments: RulePathSegment[]): string | null {
  for (let i = 0; i < pathSegments.length; i += 1) {
    const pathSegment = pathSegments[i]!;
    const ruleSegment = ruleSegments[i]!;

    if (ruleSegment.wildcard) {
      continue;
    }

    if (ruleSegment.staticString && ruleSegment.staticString !== pathSegment) {
      return null;
    }
  }

  const result: string[] = [];
  for (let i = 0; i < ruleSegments.length; i += 1) {
    const segment = ruleSegments[i]!;
    const pathSegment = pathSegments[i]!;
    if (segment.templateName) {
      result.push(`{${segment.templateName}}`);
    } else if (segment.wildcard) {
      result.push(pathSegment);
    } else {
      result.push(segment.staticString);
    }
  }

  return result.join('/');
}

function decodePathSegment(segment: string): string {
  try {
    return decodeURIComponent(segment);
  } catch {
    return segment;
  }
}

function stripQueryAndHash(raw: string): string {
  let path = raw;
  const queryIndex = path.indexOf('?');
  if (queryIndex >= 0) {
    path = path.slice(0, queryIndex);
  }
  const hashIndex = path.indexOf('#');
  if (hashIndex >= 0) {
    path = path.slice(0, hashIndex);
  }
  return path;
}

function parseAbsoluteUrl(input: string): URL | null {
  const trimmed = input.trim();
  if (!trimmed) {
    return null;
  }

  const attempts = [
    trimmed,
    /^https?:\/\//i.test(trimmed) ? null : trimmed.startsWith('//') ? `https:${trimmed}` : null,
    /^https?:\/\//i.test(trimmed) || trimmed.startsWith('//') || trimmed.startsWith('/')
      ? null
      : `https://${trimmed}`,
  ].filter((value): value is string => !!value);

  for (const candidate of attempts) {
    try {
      return new URL(candidate);
    } catch {
      continue;
    }
  }

  return null;
}

function splitPathToSegments(path: string): { segments: string[]; hadLeadingSlash: boolean } {
  if (path.trim() === '' || path.replace(/\//g, '') === '') {
    return { segments: [], hadLeadingSlash: true };
  }

  const hadLeadingSlash = path.startsWith('/');
  const normalized = hadLeadingSlash ? path : `/${path}`;
  let segments = normalized.split('/').slice(1).map(decodePathSegment);
  if (segments.at(-1) === '') {
    segments = segments.slice(0, -1);
  }

  if (!segments.length) {
    return { segments: [], hadLeadingSlash: true };
  }

  return { segments, hadLeadingSlash: true };
}

export function extractUrlPath(input: string): { segments: string[]; hadLeadingSlash: boolean } {
  const trimmed = input.trim();
  if (!trimmed) {
    return { segments: [], hadLeadingSlash: true };
  }

  const parsedUrl = parseAbsoluteUrl(trimmed);
  if (parsedUrl) {
    return splitPathToSegments(parsedUrl.pathname);
  }

  const pathOnly = stripQueryAndHash(trimmed);
  return splitPathToSegments(pathOnly);
}

export function simulateCustomTemplate(template: string, urlInput: string): TemplateSimulationResult {
  if (!urlInput.trim()) {
    return { status: 'empty_input' };
  }

  const ruleSegments = parseUserInputRuleString(template.trim());
  if (!ruleSegments.length) {
    return { status: 'no_match', message: 'Template has no path segments.' };
  }

  const { segments: pathSegments, hadLeadingSlash } = extractUrlPath(urlInput);

  if (pathSegments.length !== ruleSegments.length) {
    return {
      status: 'no_match',
      message: `Path has ${pathSegments.length} segment${pathSegments.length === 1 ? '' : 's'}; this template expects ${ruleSegments.length}.`,
    };
  }

  const joined = attemptTemplateWithRule(pathSegments, ruleSegments);
  if (!joined) {
    return {
      status: 'no_match',
      message: 'Path does not match this template (static segments must match exactly).',
    };
  }

  const outputPath = hadLeadingSlash ? `/${joined}` : joined;
  return { status: 'matched', outputPath };
}
