import { Extension } from "@tiptap/react";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import type { EditorView } from "@tiptap/pm/view";

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    slashCommands: {
      activateSlashMenu: () => ReturnType;
      insertSlash: () => ReturnType;
    };
  }
}

export interface SlashCommandState {
  active: boolean;
  query: string;
  range: { from: number; to: number } | null;
  decorationPosition: { top: number; left: number } | null;
}

const slashCommandPluginKey = new PluginKey("slashCommands");

export const SlashCommands = Extension.create({
  name: "slashCommands",

  addStorage() {
    return {
      state: {
        active: false,
        query: "",
        range: null,
        decorationPosition: null,
      } as SlashCommandState,
      onStateChange: null as ((state: SlashCommandState) => void) | null,
    };
  },

  addOptions() {
    return {
      onStateChange: undefined as
        | ((state: SlashCommandState) => void)
        | undefined,
    };
  },

  onCreate() {
    if (this.options.onStateChange) {
      this.storage.onStateChange = this.options.onStateChange;
    }
  },

  addCommands() {
    return {
      activateSlashMenu:
        () =>
        ({ view }: { view: EditorView }) => {
          const { $from } = view.state.selection;
          const textBefore = $from.parent.textContent.slice(
            0,
            $from.parentOffset,
          );
          const slashIndex = textBefore.lastIndexOf("/");
          if (slashIndex === -1) return false;

          const absolutePos = $from.start() + slashIndex;
          const coords = view.coordsAtPos(absolutePos);
          const tr = view.state.tr.setMeta(slashCommandPluginKey, {
            active: true,
            query: "",
            slashPos: absolutePos,
          });
          view.dispatch(tr);

          this.storage.onStateChange?.({
            active: true,
            query: "",
            range: { from: absolutePos, to: absolutePos + 1 },
            decorationPosition: { top: coords.bottom, left: coords.left },
          });
          return true;
        },

      insertSlash:
        () =>
        ({ view }: { view: EditorView }) => {
          const { $from } = view.state.selection;
          if ($from.parent.type.name === "codeBlock") return false;

          const insertPos = $from.pos;
          const tr = view.state.tr.insertText("/", insertPos);
          tr.setMeta(slashCommandPluginKey, {
            active: true,
            query: "",
            slashPos: insertPos,
          });
          view.dispatch(tr);

          const coords = view.coordsAtPos(insertPos);
          this.storage.onStateChange?.({
            active: true,
            query: "",
            range: { from: insertPos, to: insertPos + 1 },
            decorationPosition: { top: coords.bottom, left: coords.left },
          });
          return true;
        },
    };
  },

  addProseMirrorPlugins() {
    const storage = this.storage;

    const updateState = (newState: SlashCommandState) => {
      storage.state = newState;
      storage.onStateChange?.(newState);
    };

    return [
      new Plugin({
        key: slashCommandPluginKey,

        state: {
          init() {
            return {
              active: false,
              query: "",
              slashPos: -1,
            };
          },
          apply(tr, prev) {
            const meta = tr.getMeta(slashCommandPluginKey);
            if (meta) return meta;

            if (!prev.active) return prev;

            // If the document changed or selection moved, recheck
            if (tr.docChanged || tr.selectionSet) {
              const { $from } = tr.selection;
              const textBefore = $from.parent.textContent.slice(
                0,
                $from.parentOffset,
              );
              const slashIndex = textBefore.lastIndexOf("/");

              if (slashIndex === -1) {
                return { active: false, query: "", slashPos: -1 };
              }

              const query = textBefore.slice(slashIndex + 1);
              // Close if query has spaces (user likely typing normal text)
              if (query.includes(" ")) {
                return { active: false, query: "", slashPos: -1 };
              }

              const absoluteSlashPos =
                $from.start() + slashIndex;
              return {
                active: true,
                query,
                slashPos: absoluteSlashPos,
              };
            }

            return prev;
          },
        },

        props: {
          handleKeyDown(view: EditorView, event: KeyboardEvent) {
            const state = slashCommandPluginKey.getState(view.state);

            if (event.key === "/" && !state?.active) {
              const { $from } = view.state.selection;
              const textBefore = $from.parent.textContent.slice(
                0,
                $from.parentOffset,
              );

              // Only activate at start of line or after whitespace
              if (
                textBefore.length === 0 ||
                textBefore.endsWith(" ")
              ) {
                // Don't activate inside code blocks
                if ($from.parent.type.name === "codeBlock") {
                  return false;
                }

                setTimeout(() => {
                  const { $from: $newFrom } = view.state.selection;
                  const newTextBefore =
                    $newFrom.parent.textContent.slice(
                      0,
                      $newFrom.parentOffset,
                    );
                  const slashIdx = newTextBefore.lastIndexOf("/");
                  if (slashIdx !== -1) {
                    const absolutePos = $newFrom.start() + slashIdx;
                    const tr = view.state.tr.setMeta(
                      slashCommandPluginKey,
                      {
                        active: true,
                        query: "",
                        slashPos: absolutePos,
                      },
                    );
                    view.dispatch(tr);

                    const coords = view.coordsAtPos(absolutePos);
                    updateState({
                      active: true,
                      query: "",
                      range: {
                        from: absolutePos,
                        to: absolutePos + 1,
                      },
                      decorationPosition: {
                        top: coords.bottom,
                        left: coords.left,
                      },
                    });
                  }
                }, 0);
              }
              return false;
            }

            if (state?.active) {
              if (event.key === "Escape") {
                const tr = view.state.tr.setMeta(
                  slashCommandPluginKey,
                  {
                    active: false,
                    query: "",
                    slashPos: -1,
                  },
                );
                view.dispatch(tr);
                updateState({
                  active: false,
                  query: "",
                  range: null,
                  decorationPosition: null,
                });
                return true;
              }

              // Let arrow keys, enter pass through to the menu component
              if (
                event.key === "ArrowUp" ||
                event.key === "ArrowDown" ||
                event.key === "Enter"
              ) {
                return true;
              }
            }

            return false;
          },

          handleTextInput(view: EditorView) {
            const state = slashCommandPluginKey.getState(view.state);
            if (!state?.active) return false;

            // Update position and query on next tick
            setTimeout(() => {
              const pluginState = slashCommandPluginKey.getState(
                view.state,
              );
              if (!pluginState?.active) return;

              const { $from } = view.state.selection;
              const textBefore = $from.parent.textContent.slice(
                0,
                $from.parentOffset,
              );
              const slashIndex = textBefore.lastIndexOf("/");

              if (slashIndex === -1) {
                const tr = view.state.tr.setMeta(
                  slashCommandPluginKey,
                  {
                    active: false,
                    query: "",
                    slashPos: -1,
                  },
                );
                view.dispatch(tr);
                updateState({
                  active: false,
                  query: "",
                  range: null,
                  decorationPosition: null,
                });
                return;
              }

              const query = textBefore.slice(slashIndex + 1);
              const absoluteSlashPos = $from.start() + slashIndex;
              const coords = view.coordsAtPos(absoluteSlashPos);

              updateState({
                active: true,
                query,
                range: {
                  from: absoluteSlashPos,
                  to: $from.pos,
                },
                decorationPosition: {
                  top: coords.bottom,
                  left: coords.left,
                },
              });
            }, 0);

            return false;
          },

          handleClick(view: EditorView) {
            const state = slashCommandPluginKey.getState(view.state);
            if (state?.active) {
              const tr = view.state.tr.setMeta(
                slashCommandPluginKey,
                {
                  active: false,
                  query: "",
                  slashPos: -1,
                },
              );
              view.dispatch(tr);
              updateState({
                active: false,
                query: "",
                range: null,
                decorationPosition: null,
              });
            }
            return false;
          },
        },
      }),
    ];
  },
});
