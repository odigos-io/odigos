import { parseTemplatePath, type TemplateSegmentKind } from './template-path';
import type { AggregatedCustomTemplate } from './url-templatization-aggregate';

export type CustomTemplateTreeNode = {
  segment: string;
  kind: TemplateSegmentKind;
  children: CustomTemplateTreeNode[];
  entries: AggregatedCustomTemplate[];
};

const KIND_SORT_ORDER: Record<TemplateSegmentKind, number> = {
  static: 0,
  variable: 1,
  wildcard: 2,
};

function compareTreeNodes(a: CustomTemplateTreeNode, b: CustomTemplateTreeNode): number {
  const kindDiff = KIND_SORT_ORDER[a.kind] - KIND_SORT_ORDER[b.kind];
  if (kindDiff !== 0) {
    return kindDiff;
  }
  return a.segment.localeCompare(b.segment);
}

function sortTreeNodes(nodes: CustomTemplateTreeNode[]): void {
  nodes.sort(compareTreeNodes);
  for (const node of nodes) {
    sortTreeNodes(node.children);
  }
}

function findOrCreateChild(
  level: CustomTemplateTreeNode[],
  segment: string,
  kind: TemplateSegmentKind,
): CustomTemplateTreeNode {
  const existing = level.find((node) => node.segment === segment && node.kind === kind);
  if (existing) {
    return existing;
  }
  const created: CustomTemplateTreeNode = {
    segment,
    kind,
    children: [],
    entries: [],
  };
  level.push(created);
  return created;
}

export function buildCustomTemplateTree(items: AggregatedCustomTemplate[]): CustomTemplateTreeNode[] {
  const roots: CustomTemplateTreeNode[] = [];

  for (const item of items) {
    const segments = parseTemplatePath(item.template);
    if (segments.length === 0) {
      findOrCreateChild(roots, item.template, 'static').entries.push(item);
      continue;
    }

    let level = roots;
    for (let index = 0; index < segments.length; index += 1) {
      const { kind, text } = segments[index];
      const node = findOrCreateChild(level, text, kind);
      if (index === segments.length - 1) {
        node.entries.push(item);
      }
      level = node.children;
    }
  }

  sortTreeNodes(roots);
  return roots;
}
