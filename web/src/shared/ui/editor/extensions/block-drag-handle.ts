import { Extension } from "@tiptap/react";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import type { EditorView } from "@tiptap/pm/view";

export interface DraggingState {
  from: number;
  to: number;
  nodeJSON: Record<string, unknown>;
}

const blockDragHandleKey = new PluginKey("blockDragHandle");

export const BlockDragHandle = Extension.create({
  name: "blockDragHandle",

  addStorage() {
    return {
      dragging: null as DraggingState | null,
    };
  },

  addProseMirrorPlugins() {
    const storage = this.storage;

    return [
      new Plugin({
        key: blockDragHandleKey,
        props: {
          handleDOMEvents: {
            dragenter(_view: EditorView, event: DragEvent) {
              if (storage.dragging) {
                event.preventDefault();
                return true;
              }
              return false;
            },
            dragover(_view: EditorView, event: DragEvent) {
              if (storage.dragging) {
                event.preventDefault();
                event.dataTransfer!.dropEffect = "move";
                return true;
              }
              return false;
            },
            // Handle drop in DOM events to bypass PM's editHandlers.drop
            // (PM checks posAtCoords before calling props.handleDrop — our handler never ran)
            drop(view: EditorView, event: DragEvent) {
              const dragging = storage.dragging;
              if (!dragging) return false;

              event.preventDefault();
              storage.dragging = null;
              view.dragging = null;

              const coords = { left: event.clientX, top: event.clientY };
              const dropPos = view.posAtCoords(coords);
              if (!dropPos) return true;

              const $drop = view.state.doc.resolve(dropPos.pos);
              if ($drop.depth === 0) return true;
              let targetPos = $drop.before(1);
              const targetNode = view.state.doc.nodeAt(targetPos);
              if (targetNode) {
                const targetDOM = view.nodeDOM(targetPos);
                if (targetDOM instanceof HTMLElement) {
                  const rect = targetDOM.getBoundingClientRect();
                  if (event.clientY > rect.top + rect.height / 2) {
                    targetPos = targetPos + targetNode.nodeSize;
                  }
                }
              }

              if (targetPos >= dragging.from && targetPos <= dragging.to) return true;

              const node = view.state.schema.nodeFromJSON(dragging.nodeJSON);
              const { tr } = view.state;

              if (targetPos <= dragging.from) {
                tr.insert(targetPos, node);
                const mappedFrom = tr.mapping.map(dragging.from);
                const mappedTo = tr.mapping.map(dragging.to);
                tr.delete(mappedFrom, mappedTo);
              } else {
                tr.delete(dragging.from, dragging.to);
                const mappedTarget = tr.mapping.map(targetPos);
                tr.insert(mappedTarget, node);
              }

              view.dispatch(tr);
              return true;
            },
          },
        },
      }),
    ];
  },
});
