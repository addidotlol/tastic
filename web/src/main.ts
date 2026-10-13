import {
  FileDiff,
  getFiletypeFromFileName,
  parsePatchFiles,
  type FileDiffMetadata,
} from "@pierre/diffs";
import { isBundledLanguage } from "./languages";
import "./style.css";

type SpanList = [startByte: number, endByte: number][];

interface DifftasticFile {
  path: string;
  language: string;
  status: string;
  deletions: Record<number, SpanList>;
  additions: Record<number, SpanList>;
}

interface ViewerData {
  repository: string;
  branch: string;
  range: string;
  generatedAt: string;
  patch: string;
  files: DifftasticFile[];
}

type DiffStyle = "split" | "unified";

const data: ViewerData = JSON.parse(
  document.getElementById("tastic-data")?.textContent ?? "null",
);

const difftasticByPath = new Map(data.files.map((file) => [file.path, file]));

function findDifftastic(fileDiff: FileDiffMetadata) {
  return (
    difftasticByPath.get(fileDiff.name) ??
    (fileDiff.prevName ? difftasticByPath.get(fileDiff.prevName) : undefined)
  );
}

function attachDifftastic(fileDiff: FileDiffMetadata): FileDiffMetadata {
  const difftastic = findDifftastic(fileDiff);
  const language = fileDiff.lang ?? getFiletypeFromFileName(fileDiff.name);
  return {
    ...fileDiff,
    lang: isBundledLanguage(language) ? language : "text",
    difftastic: difftastic && {
      deletions: difftastic.deletions,
      additions: difftastic.additions,
    },
  };
}

function lineTotals(fileDiff: FileDiffMetadata) {
  return fileDiff.hunks.reduce(
    (totals, hunk) => ({
      additions: totals.additions + hunk.additionLines,
      deletions: totals.deletions + hunk.deletionLines,
    }),
    { additions: 0, deletions: 0 },
  );
}

function element<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  props: Partial<HTMLElementTagNameMap[K]> = {},
  children: (Node | string)[] = [],
) {
  const node = Object.assign(document.createElement(tag), props);
  node.append(...children);
  return node;
}

function totalsBadge({ additions, deletions }: { additions: number; deletions: number }) {
  return element("span", { className: "totals" }, [
    element("span", { className: "added", textContent: `+${additions}` }),
    element("span", { className: "removed", textContent: `−${deletions}` }),
  ]);
}

function languageBadge(fileDiff: FileDiffMetadata) {
  const difftastic = findDifftastic(fileDiff);
  if (!difftastic) return null;
  const structural = difftastic.language !== "Text";
  return element("span", {
    className: structural ? "badge structural" : "badge",
    textContent: structural ? difftastic.language : "line diff",
    title: structural
      ? `difftastic parsed this file as ${difftastic.language}`
      : "difftastic has no parser for this file and fell back to a line diff",
  });
}

function slugFor(index: number) {
  return `file-${index + 1}`;
}

function renderSummary(fileDiffs: FileDiffMetadata[]) {
  const totals = fileDiffs.map(lineTotals).reduce(
    (sum, next) => ({
      additions: sum.additions + next.additions,
      deletions: sum.deletions + next.deletions,
    }),
    { additions: 0, deletions: 0 },
  );

  const title = element("h1", {}, [
    element("span", { className: "repository", textContent: data.repository }),
    element("span", { className: "branch", textContent: data.branch }),
  ]);
  const meta = element("p", { className: "meta" }, [
    `git diff ${data.range || "(working tree)"} · ${fileDiffs.length} file${fileDiffs.length === 1 ? "" : "s"} · `,
    totalsBadge(totals),
    ` · ${new Date(data.generatedAt).toLocaleString()}`,
  ]);

  const nav = element(
    "ol",
    { className: "file-list" },
    fileDiffs.map((fileDiff, index) =>
      element("li", {}, [
        element("a", { href: `#${slugFor(index)}`, textContent: fileDiff.name }),
        totalsBadge(lineTotals(fileDiff)),
      ]),
    ),
  );

  return { title, meta, nav };
}

function styleToggle(onChange: (style: DiffStyle) => void, initial: DiffStyle) {
  const buttons = (["split", "unified"] as const).map((style) =>
    element("button", {
      type: "button",
      textContent: style,
      ariaPressed: String(style === initial),
      onclick: () => {
        for (const button of buttons) button.ariaPressed = String(button === buttonFor(style));
        onChange(style);
      },
    }),
  );
  const buttonFor = (style: DiffStyle) => buttons[style === "split" ? 0 : 1];
  return element("div", { className: "toggle", role: "group" }, buttons);
}

function initialStyle(): DiffStyle {
  if (location.hash === "#unified") return "unified";
  return window.matchMedia("(max-width: 900px)").matches ? "unified" : "split";
}

function main() {
  const root = document.getElementById("app");
  if (!root) return;

  const fileDiffs = parsePatchFiles(data.patch)
    .flatMap((patch) => patch.files)
    .map(attachDifftastic);

  let diffStyle = initialStyle();
  const viewers: FileDiff[] = [];

  const { title, meta, nav } = renderSummary(fileDiffs);
  const toggle = styleToggle((style) => {
    diffStyle = style;
    for (const viewer of viewers) viewer.setOptions({ ...viewer.options, diffStyle });
    for (const viewer of viewers) viewer.rerender();
  }, diffStyle);

  root.append(
    element("header", { className: "summary" }, [
      element("div", { className: "heading" }, [title, toggle]),
      meta,
      nav,
    ]),
  );

  if (fileDiffs.length === 0) {
    root.append(element("p", { className: "empty", textContent: "No changes." }));
    return;
  }

  fileDiffs.forEach((fileDiff, index) => {
    const section = element("section", { id: slugFor(index), className: "file" });
    root.append(section);
    const viewer = new FileDiff({
      theme: { dark: "pierre-dark", light: "pierre-light" },
      themeType: "system",
      diffStyle,
      diffIndicators: "bars",
      hunkSeparators: "line-info",
      overflow: "scroll",
      renderHeaderMetadata: languageBadge,
    });
    viewers.push(viewer);
    viewer.render({ fileDiff, containerWrapper: section });
  });
}

main();
