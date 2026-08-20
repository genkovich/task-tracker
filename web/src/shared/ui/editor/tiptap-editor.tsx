import { useEditor, EditorContent, type Editor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Link from "@tiptap/extension-link";
import Placeholder from "@tiptap/extension-placeholder";
import CodeBlockLowlight from "@tiptap/extension-code-block-lowlight";
import Image from "@tiptap/extension-image";
import { common, createLowlight } from "lowlight";
import { Markdown } from "tiptap-markdown";
import { ReactNodeViewRenderer } from "@tiptap/react";
import { useRef, useState, useCallback } from "react";
import { cn } from "@/shared/lib/utils";
import { Code2 } from "lucide-react";

import { SlashCommands } from "./extensions/slash-commands";
import type { SlashCommandState } from "./extensions/slash-commands";
import { BlockDragHandle } from "./extensions/block-drag-handle";
import { BubbleToolbar } from "./components/bubble-toolbar";
import { SlashCommandMenu } from "./components/slash-command-menu";
import { BlockControls } from "./components/block-controls";
import { EditorToolbar } from "./components/editor-toolbar";
import { CodeBlockView } from "./components/code-block-view";

import "highlight.js/styles/github-dark.min.css";

const lowlight = createLowlight(common);

interface TiptapEditorProps {
  content: string;
  onUpdate: (md: string) => void;
  placeholder?: string;
  className?: string;
  minHeight?: string;
}

const CustomCodeBlock = CodeBlockLowlight.extend({
  addNodeView() {
    return ReactNodeViewRenderer(CodeBlockView);
  },
});

export function TiptapEditor({
  content,
  onUpdate,
  placeholder = "Type '/' for commands...",
  className,
  minHeight = "100px",
}: TiptapEditorProps) {
  const onUpdateRef = useRef(onUpdate);
  onUpdateRef.current = onUpdate;

  const [showSource, setShowSource] = useState(false);
  const [sourceText, setSourceText] = useState("");
  const [isHovered, setIsHovered] = useState(false);
  const [slashState, setSlashState] = useState<SlashCommandState>({
    active: false,
    query: "",
    range: null,
    decorationPosition: null,
  });

  const editor = useEditor({
    extensions: [
      StarterKit.configure({ codeBlock: false, dropcursor: false }),
      Link.configure({
        openOnClick: false,
        protocols: ["http", "https", "mailto"],
        validate: (href) => /^https?:\/\/|^mailto:/i.test(href),
      }),
      CustomCodeBlock.configure({ lowlight }),
      Image.configure({
        HTMLAttributes: {
          class: "max-w-full rounded-lg my-4",
        },
      }),
      Placeholder.configure({ placeholder }),
      Markdown.configure({
        html: false,
        transformPastedText: true,
        transformCopiedText: true,
      }),
      SlashCommands.configure({
        onStateChange: (state: SlashCommandState) => {
          setSlashState(state);
        },
      }),
      BlockDragHandle,
    ],
    content,
    immediatelyRender: false,
    onUpdate: ({ editor }) => {
      const md = (editor.storage as Record<string, any>).markdown; // eslint-disable-line @typescript-eslint/no-explicit-any
      onUpdateRef.current(md.getMarkdown());
    },
  });

  const handleToggleSource = useCallback(() => {
    if (!editor) return;

    if (!showSource) {
      const md = (editor.storage as Record<string, any>).markdown; // eslint-disable-line @typescript-eslint/no-explicit-any
      setSourceText(md.getMarkdown());
    } else {
      editor.commands.setContent(sourceText);
      const md = (editor.storage as Record<string, any>).markdown; // eslint-disable-line @typescript-eslint/no-explicit-any
      onUpdateRef.current(md.getMarkdown());
    }

    setShowSource(!showSource);
  }, [editor, showSource, sourceText]);

  const handleSourceChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setSourceText(e.target.value);
    onUpdateRef.current(e.target.value);
  }, []);

  const handleCloseSlash = useCallback(() => {
    setSlashState({
      active: false,
      query: "",
      range: null,
      decorationPosition: null,
    });
  }, []);

  return (
    <div
      className={cn("relative group", className)}
      style={{ "--editor-min-h": minHeight } as React.CSSProperties}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      {/* Markdown source toggle */}
      <button
        type="button"
        onClick={handleToggleSource}
        title="Markdown source"
        className={cn(
          "absolute right-2 top-2 z-10 inline-flex items-center justify-center h-7 w-7 rounded text-muted-foreground hover:text-foreground hover:bg-muted transition-all",
          showSource
            ? "opacity-100 bg-muted text-foreground"
            : isHovered
              ? "opacity-100"
              : "opacity-0",
        )}
      >
        <Code2 size={16} />
      </button>

      {showSource ? (
        <textarea
          value={sourceText}
          onChange={handleSourceChange}
          style={{ minHeight }}
          className="w-full p-3 bg-transparent font-mono text-sm outline-none resize-y rounded-md border"
          placeholder={placeholder}
        />
      ) : (
        <>
          {editor && <BubbleToolbar editor={editor} />}
          {editor && <BlockControls editor={editor} />}
          {editor && (
            <SlashCommandMenu editor={editor} state={slashState} onClose={handleCloseSlash} />
          )}
          <EditorContent
            editor={editor}
            className={cn(
              "prose prose-sm dark:prose-invert max-w-none",
              // Base editor styles
              "[&_.tiptap]:outline-none [&_.tiptap]:min-h-[var(--editor-min-h)] [&_.tiptap]:py-2",
              // Placeholder
              "[&_.tiptap_p.is-editor-empty:first-child::before]:text-muted-foreground [&_.tiptap_p.is-editor-empty:first-child::before]:content-[attr(data-placeholder)] [&_.tiptap_p.is-editor-empty:first-child::before]:float-left [&_.tiptap_p.is-editor-empty:first-child::before]:h-0 [&_.tiptap_p.is-editor-empty:first-child::before]:pointer-events-none",
              // Headings
              "[&_.tiptap_h1]:text-2xl [&_.tiptap_h1]:font-bold [&_.tiptap_h1]:mt-6 [&_.tiptap_h1]:mb-3",
              "[&_.tiptap_h2]:text-xl [&_.tiptap_h2]:font-semibold [&_.tiptap_h2]:mt-5 [&_.tiptap_h2]:mb-2",
              "[&_.tiptap_h3]:text-lg [&_.tiptap_h3]:font-semibold [&_.tiptap_h3]:mt-4 [&_.tiptap_h3]:mb-2",
              // Blockquote
              "[&_.tiptap_blockquote]:border-l-4 [&_.tiptap_blockquote]:border-primary/30 [&_.tiptap_blockquote]:pl-4 [&_.tiptap_blockquote]:italic [&_.tiptap_blockquote]:text-muted-foreground",
              // Images
              "[&_.tiptap_img]:max-w-full [&_.tiptap_img]:rounded-lg [&_.tiptap_img]:my-4",
              // Horizontal rule
              "[&_.tiptap_hr]:border-border [&_.tiptap_hr]:my-6",
              // Lists
              "[&_.tiptap_ul]:list-disc [&_.tiptap_ul]:pl-6 [&_.tiptap_ol]:list-decimal [&_.tiptap_ol]:pl-6",
              "[&_.tiptap_li]:my-0.5",
              // Selected node (drag)
              "[&_.ProseMirror-selectednode]:outline-none [&_.ProseMirror-selectednode]:bg-primary/10 [&_.ProseMirror-selectednode]:rounded [&_.ProseMirror-selectednode]:ring-2 [&_.ProseMirror-selectednode]:ring-primary/30",
            )}
          />
          {editor && (
            <div className="opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity pb-2">
              <EditorToolbar editor={editor} />
            </div>
          )}
        </>
      )}
    </div>
  );
}
