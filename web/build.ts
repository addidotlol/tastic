import type { BunPlugin } from "bun";

const patchedPierreFile = Bun.resolveSync(
  "./node_modules/@pierre/diffs/dist/utils/renderDiffWithHighlighter.js",
  import.meta.dir,
);
if (!(await Bun.file(patchedPierreFile).text()).includes("pushDifftasticDecorations")) {
  console.error(
    "@pierre/diffs is missing tastic's difftastic patch. Install with `bun install` in web/ (npm and pnpm don't apply bun's patchedDependencies).",
  );
  process.exit(1);
}

const themingFile = (relativePath: string) =>
  JSON.stringify(
    Bun.resolveSync(`./node_modules/@pierre/theming/dist/${relativePath}`, import.meta.dir),
  );

const pierreThemesOnlyModule = `
import { createThemeCollection } from ${themingFile("modules/createThemeCollection.js")};
import { pierreThemes } from ${themingFile("collections/pierre.js")};
export { createTheme } from ${themingFile("modules/createTheme.js")};
export { pierreThemes };
export const shikiThemes = createThemeCollection({ themes: [] });
export const themes = createThemeCollection({ themes: [pierreThemes] });
`;

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
      contents: pierreThemesOnlyModule,
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
