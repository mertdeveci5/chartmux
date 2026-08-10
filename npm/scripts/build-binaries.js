import { spawnSync } from "node:child_process";
import { chmodSync, copyFileSync, mkdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const repositoryRoot = join(packageRoot, "..");
const vendor = join(packageRoot, "vendor");
const manifest = JSON.parse(readFileSync(join(packageRoot, "package.json"), "utf8"));
const targets = [
  ["darwin", "arm64", "chartmux-darwin-arm64"],
  ["darwin", "amd64", "chartmux-darwin-x64"],
  ["linux", "arm64", "chartmux-linux-arm64"],
  ["linux", "amd64", "chartmux-linux-x64"],
  ["windows", "arm64", "chartmux-win32-arm64.exe"],
  ["windows", "amd64", "chartmux-win32-x64.exe"]
];

mkdirSync(vendor, { recursive: true });
copyFileSync(join(repositoryRoot, "LICENSE"), join(packageRoot, "LICENSE"));

for (const [goos, goarch, filename] of targets) {
  const output = join(vendor, filename);
  const result = spawnSync(
    "go",
    ["build", "-trimpath", "-ldflags", `-s -w -X main.version=${manifest.version}`, "-o", output, "./cmd/chartmux"],
    {
      cwd: repositoryRoot,
      env: { ...process.env, CGO_ENABLED: "0", GOOS: goos, GOARCH: goarch },
      stdio: "inherit"
    }
  );
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
  if (goos !== "windows") {
    chmodSync(output, 0o755);
  }
}
