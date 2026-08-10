#!/usr/bin/env node

import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const platforms = new Set(["darwin", "linux", "win32"]);
const architectures = new Set(["arm64", "x64"]);

if (!platforms.has(process.platform) || !architectures.has(process.arch)) {
  console.error(`chartmux does not provide a binary for ${process.platform}/${process.arch}`);
  process.exit(1);
}

const extension = process.platform === "win32" ? ".exe" : "";
const filename = `chartmux-${process.platform}-${process.arch}${extension}`;
const binary = join(dirname(fileURLToPath(import.meta.url)), "..", "vendor", filename);

if (!existsSync(binary)) {
  console.error(`chartmux package is missing ${filename}; reinstall the package`);
  process.exit(1);
}

const child = spawn(binary, process.argv.slice(2), { stdio: "inherit" });

child.on("error", (error) => {
  console.error(`chartmux failed to start: ${error.message}`);
  process.exit(1);
});

child.on("exit", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 1);
});
