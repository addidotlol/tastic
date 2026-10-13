import type { BunPlugin } from "bun";

const pierreCollectionPath = Bun.resolveSync(
  "./node_modules/@pierre/theming/dist/collections/pierre.js",
  import.meta.dir,
);

const trimBundle: BunPlugin = {
  name: "tastic-bundle-trim",
  setup(build) {
    build.onResolve({ filter: /^shiki$/ }, () => ({
      path: Bun.resolveSync("./src/shiki-shim.ts", import.meta.dir),
    }));
    build.onResolve({ filter: /^@pierre\/theming\/themes$/ }, () => ({
      path: "pierre-themes-only",
      namespace: "tastic-pierre-themes",
    }));
    build.onLoad({ filter: /.*/, namespace: "tastic-pierre-themes" }, () => ({
      contents: [
        `export { pierreThemes } from ${JSON.stringify(pierreCollectionPath)};`,
        "export const shikiThemes = { getTheme: () => undefined };",
      ].join("\n"),
      loader: "js",
    }));
    build.onResolve({ filter: /^shiki\/wasm$/ }, () => ({
      path: "shiki-wasm-stub",
      namespace: "tastic-stub",
    }));
    build.onLoad({ filter: /.*/, namespace: "tastic-stub" }, () => ({
      contents: "export default undefined;",
      loader: "js",
    }));
  },
};

const outdir = process.argv[2] ?? "../assets";

const result = await Bun.build({
  entrypoints: ["./src/main.ts"],
  outdir,
  naming: "viewer.[ext]",
  target: "browser",
  format: "esm",
  minify: true,
  plugins: [trimBundle],
});

if (!result.success) {
  for (const log of result.logs) console.error(log);
  process.exit(1);
}

for (const output of result.outputs) {
  console.log(`${output.path} ${(output.size / 1024).toFixed(0)} KiB`);
}
