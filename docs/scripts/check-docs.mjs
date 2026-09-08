// Source-only checks; no license key, network, or Mintlify account is required.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const config = JSON.parse(fs.readFileSync(path.join(root, 'docs.json')));
const requireExport = process.argv.includes('--export');
const products = ['oss', 'enterprise', 'central', 'vmagent', 'cloud-connectors'].filter((name) =>
  fs.existsSync(path.join(root, name)),
);
const errors = [];
const fail = (message) => errors.push(message);
const exists = (name) => fs.existsSync(path.join(root, name));
const walk = (dir) =>
  fs
    .readdirSync(path.join(root, dir), { recursive: true })
    .filter((file) => file.endsWith('.mdx'))
    .map((file) => dir + '/' + file);
const pages = new Set();
function navigation(value, key) {
  if (typeof value === 'string' && key === 'pages' && !value.startsWith('https://'))
    pages.add(value);
  else if (Array.isArray(value)) value.forEach((item) => navigation(item, key));
  else if (value && typeof value === 'object')
    Object.entries(value).forEach(([key, item]) => navigation(item, key));
}
navigation(config.navigation);
for (const page of pages) {
  if (!exists(page + '.mdx')) fail(`Navigation page missing: ${page}`);
}
const source = [...products.flatMap(walk), ...walk('snippets')];
for (const file of source) {
  if (requireExport && !file.startsWith('snippets/') && !pages.has(file.slice(0, -4)))
    fail(`Export would omit page (add hidden navigation group): ${file}`);
  const text = fs.readFileSync(path.join(root, file), 'utf8');
  for (const match of text.matchAll(/\bimport\s+[^;\n]*?\s+from\s+["']([^"']+)["']/g)) {
    const target = match[1];
    if (!target.startsWith('/') && !target.startsWith('.')) continue;
    const resolved = target.startsWith('/')
      ? target.slice(1)
      : path.posix.normalize(path.posix.join(path.posix.dirname(file), target));
    if (!exists(resolved)) fail(`${file}: missing import ${target}`);
  }
  for (const match of text.matchAll(/(?:["'(])(\/images\/[^"')\s?#]+)(?:["')?#])/g)) {
    if (!exists(decodeURI(match[1]))) fail(`${file}: missing image ${match[1]}`);
  }
}
if (errors.length) {
  console.error(errors.join('\n'));
  process.exit(1);
}
console.log(`Checked ${source.length} MDX files and ${pages.size} navigation pages.`);
