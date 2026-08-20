import type { Editor } from "@tiptap/react";
import { Undo2, Redo2, Link2 } from "lucide-react";
import { useCallback } from "react";
import { cn } from "@/shared/lib/utils";

interface ToolbarButtonProps {
  onClick: () => void;
  disabled?: boolean;
  active?: boolean;
  title: string;
  children: React.ReactNode;
}

function ToolbarButton({
  onClick,
  disabled,
  active,
  title,
  children,
}: ToolbarButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      title={title}
      className={cn(
        "flex items-center justify-center h-7 w-7 rounded text-muted-foreground hover:text-foreground hover:bg-muted transition-colors disabled:opacity-30 disabled:pointer-events-none",
        active && "bg-muted text-foreground",
      )}
    >
      {children}
    </button>
  );
}

export function EditorToolbar({ editor }: { editor: Editor }) {
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
    editor.chain().focus().extendMarkRange("link").setLink({ href: url }).run();
  }, [editor]);

  return (
    <div className="flex items-center gap-0.5 pt-1 border-t border-border/50">
      <ToolbarButton
        onClick={() => editor.chain().focus().undo().run()}
        disabled={!editor.can().undo()}
        title="Undo"
      >
        <Undo2 size={14} />
      </ToolbarButton>
      <ToolbarButton
        onClick={() => editor.chain().focus().redo().run()}
        disabled={!editor.can().redo()}
        title="Redo"
      >
        <Redo2 size={14} />
      </ToolbarButton>
      <div className="w-px h-4 bg-border/50 mx-1" />
      <ToolbarButton
        onClick={handleLink}
        active={editor.isActive("link")}
        title="Insert link"
      >
        <Link2 size={14} />
      </ToolbarButton>
    </div>
  );
}
