import { useEffect, useRef, useCallback } from "react";
import type { Editor } from "@tiptap/react";
import {
  Command,
  CommandList,
  CommandGroup,
  CommandItem,
  CommandEmpty,
} from "@/shared/ui/command";
import type { SlashCommandState } from "../extensions/slash-commands";
import { SLASH_COMMANDS, type SlashCommandItem } from "../config/commands";

interface SlashCommandMenuProps {
  editor: Editor;
  state: SlashCommandState;
  onClose: () => void;
}

export function SlashCommandMenu({
  editor,
  state,
  onClose,
}: SlashCommandMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);

  const filteredCommands = SLASH_COMMANDS.filter((cmd) => {
    if (!state.query) return true;
    const q = state.query.toLowerCase();
    return (
      cmd.title.toLowerCase().includes(q) ||
      cmd.aliases.some((a) => a.includes(q))
    );
  });

  const executeCommand = useCallback(
    (item: SlashCommandItem) => {
      if (!state.range) return;

      // Delete the /query text
      editor
        .chain()
        .focus()
        .deleteRange({ from: state.range.from, to: state.range.to + 1 })
        .run();

      // Execute the command
      item.command(editor);
      onClose();
    },
    [editor, state.range, onClose],
  );

  // Handle keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!state.active) return;

      if (e.key === "ArrowUp" || e.key === "ArrowDown" || e.key === "Enter") {
        // Forward to cmdk
        const cmdkInput = menuRef.current?.querySelector("[cmdk-input]");
        if (cmdkInput) {
          cmdkInput.dispatchEvent(
            new KeyboardEvent("keydown", {
              key: e.key,
              bubbles: true,
            }),
          );
        }
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [state.active]);

  // Close on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [onClose]);

  if (!state.active || !state.decorationPosition) return null;

  if (filteredCommands.length === 0) {
    // Auto-close when no results
    return null;
  }

  return (
    <div
      ref={menuRef}
      data-slash-menu
      className="fixed z-50 w-64 rounded-lg border bg-popover shadow-lg"
      style={{
        top: state.decorationPosition.top + 4,
        left: state.decorationPosition.left,
      }}
    >
      <Command
        filter={(value, search) => {
          // We handle filtering ourselves based on the editor query
          if (!search) return 1;
          const q = search.toLowerCase();
          const cmd = SLASH_COMMANDS.find(
            (c) => c.title.toLowerCase() === value.toLowerCase(),
          );
          if (!cmd) return 0;
          if (cmd.title.toLowerCase().includes(q)) return 1;
          if (cmd.aliases.some((a) => a.includes(q))) return 1;
          return 0;
        }}
      >
        <CommandList>
          <CommandEmpty>No results</CommandEmpty>
          <CommandGroup heading="Blocks">
            {filteredCommands.map((item) => (
              <CommandItem
                key={item.title}
                value={item.title}
                onSelect={() => executeCommand(item)}
              >
                <item.icon className="size-4 text-muted-foreground" />
                <div className="flex flex-col">
                  <span className="text-sm">{item.title}</span>
                  <span className="text-xs text-muted-foreground">
                    {item.description}
                  </span>
                </div>
              </CommandItem>
            ))}
          </CommandGroup>
        </CommandList>
      </Command>
    </div>
  );
}
