import { NodeViewContent, NodeViewWrapper } from "@tiptap/react";
import type { NodeViewProps } from "@tiptap/react";
import { CODE_LANGUAGES } from "../config/languages";

export function CodeBlockView({
  node,
  updateAttributes,
}: NodeViewProps) {
  const language = (node.attrs.language as string) ?? "";

  return (
    <NodeViewWrapper className="relative my-3">
      <select
        contentEditable={false}
        value={language}
        onChange={(e) => updateAttributes({ language: e.target.value })}
        className="absolute right-2 top-2 z-10 rounded bg-white/10 px-2 py-0.5 text-xs text-neutral-400 outline-none hover:text-neutral-200 cursor-pointer border-none"
      >
        {CODE_LANGUAGES.map((lang) => (
          <option key={lang.value} value={lang.value} className="bg-neutral-900 text-neutral-200">
            {lang.label}
          </option>
        ))}
      </select>
      <pre
        className="rounded-lg bg-[#0d1117] p-4 pt-10 overflow-x-auto"
        spellCheck={false}
      >
        <NodeViewContent
          as={"code" as "div"}
          className={[
            "text-sm bg-transparent p-0",
            language && `language-${language}`,
          ].filter(Boolean).join(" ")}
        />
      </pre>
    </NodeViewWrapper>
  );
}
