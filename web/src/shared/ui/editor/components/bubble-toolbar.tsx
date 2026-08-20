import type { Editor } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import { Bold, Italic, Strikethrough, Code, Link2 } from "lucide-react";
import { useCallback } from "react";
import { cn } from "@/shared/lib/utils";

function BubbleButton({
  onClick,
  active,
  title,
  children,
}: {
  onClick: () => void;
  active?: boolean;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={title}
      className={cn(
        "inline-flex items-center justify-center h-8 w-8 rounded text-neutral-300 hover:text-white hover:bg-white/10 transition-colors",
        active && "text-white bg-white/15",
      )}
    >
      {children}
    </button>
  );
}

export function BubbleToolbar({ editor }: { editor: Editor }) {
  const handleLink = useCallback(() => {
    const isActive = editor.isActive("link");
    const currentUrl = isActive
      ? (editor.getAttributes("link").href as string)
      : "";

    const url = window.prompt("URL:", currentUrl);
    if (url === null) return;

    if (url === "") {
      editor.chain().focus().extendMarkRange("link").unsetLink().run();
      return;
    }

    editor
      .chain()
      .focus()
      .extendMarkRange("link")
      .setLink({ href: url })
      .run();
  }, [editor]);

  return (
    <BubbleMenu
      editor={editor}
      options={{
        placement: "top",
        offset: 8,
      }}
      shouldShow={({ editor, state }) => {
        const { from, to } = state.selection;
        if (from === to) return false;
        if (editor.isActive("codeBlock")) return false;
        if (editor.isActive("image")) return false;
        return true;
      }}
    >
      <div className="flex items-center gap-0.5 rounded-lg bg-neutral-900 border border-neutral-700 shadow-xl px-1 py-0.5">
        <BubbleButton
          onClick={() => editor.chain().focus().toggleBold().run()}
          active={editor.isActive("bold")}
          title="Bold"
        >
          <Bold size={15} />
        </BubbleButton>
        <BubbleButton
          onClick={() => editor.chain().focus().toggleItalic().run()}
          active={editor.isActive("italic")}
          title="Italic"
        >
          <Italic size={15} />
        </BubbleButton>
        <BubbleButton
          onClick={() => editor.chain().focus().toggleStrike().run()}
          active={editor.isActive("strike")}
          title="Strikethrough"
        >
          <Strikethrough size={15} />
        </BubbleButton>
        <BubbleButton
          onClick={() => editor.chain().focus().toggleCode().run()}
          active={editor.isActive("code")}
          title="Code"
        >
          <Code size={15} />
        </BubbleButton>
        <BubbleButton
          onClick={handleLink}
          active={editor.isActive("link")}
          title="Link"
        >
          <Link2 size={15} />
        </BubbleButton>
      </div>
    </BubbleMenu>
  );
}
