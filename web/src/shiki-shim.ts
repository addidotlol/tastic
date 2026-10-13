import {
  codeToHtml,
  createCssVariablesTheme,
  createHighlighterCore,
  getTokenStyleObject,
  stringifyTokenStyle,
  type HighlighterCoreOptions,
} from "shiki/core";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";

export { bundledLanguages } from "./languages";
export {
  codeToHtml,
  createCssVariablesTheme,
  createJavaScriptRegexEngine,
  getTokenStyleObject,
  stringifyTokenStyle,
};

export function createOnigurumaEngine(): never {
  throw new Error("tastic only bundles the JavaScript regex engine");
}

type ShorthandOptions = Omit<HighlighterCoreOptions, "langs" | "themes"> & {
  langs?: unknown[];
  themes?: unknown[];
};

const dropNames = <T>(entries: unknown[] = []) =>
  entries.filter((entry) => typeof entry !== "string") as T[];

export function createHighlighter({ langs, themes, ...options }: ShorthandOptions) {
  return createHighlighterCore({
    ...options,
    langs: dropNames(langs),
    themes: dropNames(themes),
  });
}
