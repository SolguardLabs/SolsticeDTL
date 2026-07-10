import { readdirSync } from "node:fs";
import { join, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const root = resolve(fileURLToPath(new URL("..", import.meta.url)));

function run(command, args) {
  const result = spawnSync(command, args, {
    cwd: root,
    encoding: "utf8",
    stdio: "inherit",
    shell: false,
  });

  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

const testFiles = readdirSync(join(root, "tests", "node"))
  .filter((file) => file.endsWith(".test.ts"))
  .sort()
  .map((file) => join("tests", "node", file));

run("gofmt", ["-w", "src"]);
run("git", ["diff", "--exit-code", "--", "src"]);
run("go", ["test", "./..."]);
run("go", ["vet", "./..."]);
run(process.execPath, ["scripts/build.mjs"]);
run(process.execPath, ["--test", "--experimental-strip-types", ...testFiles]);
