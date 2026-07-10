import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const src = join(root, "src");
const min = Number(process.argv[2] ?? 3000);
const max = Number(process.argv[3] ?? 4000);

function walk(dir) {
  const files = [];
  for (const entry of readdirSync(dir)) {
    const path = join(dir, entry);
    const stat = statSync(path);
    if (stat.isDirectory()) {
      files.push(...walk(path));
    } else {
      files.push(path);
    }
  }
  return files;
}

const lines = walk(src).reduce((total, file) => {
  return total + readFileSync(file, "utf8").split(/\r?\n/).length;
}, 0);

if (lines < min || lines > max) {
  console.error(`src LOC ${lines} outside expected range ${min}-${max}`);
  process.exit(1);
}

console.log(`src LOC ${lines}`);
