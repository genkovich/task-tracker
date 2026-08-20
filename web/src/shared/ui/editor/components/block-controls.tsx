import type { Editor } from "@tiptap/react";
import { useEffect, useRef, useState, useCallback } from "react";
import { createPortal } from "react-dom";
import { GripVertical, Plus } from "lucide-react";

interface BlockInfo {
  pos: number;
  node: ReturnType<Editor["state"]["doc"]["nodeAt"]>;
  rect: DOMRect;
}

export function BlockControls({ editor }: { editor: Editor }) {
  const [activeBlock, setActiveBlock] = useState<BlockInfo | null>(null);
  const [visible, setVisible] = useState(false);
  const [dropIndicator, setDropIndicator] = useState<{ top: number; left: number; width: number } | null>(null);
  const controlsRef = useRef<HTMLDivElement>(null);
  const hideTimeoutRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const rafRef = useRef<number>(undefined);
  const isDraggingRef = useRef(false);
  const dragGhostRef = useRef<HTMLDivElement | null>(null);
  const lastIndicatorTopRef = useRef<number | null>(null);
  const lastGapPosRef = useRef<number | null>(null);

  const showControls = useCallback((block: BlockInfo) => {
    clearTimeout(hideTimeoutRef.current);
    setActiveBlock(block);
    setVisible(true);
  }, []);

  const scheduleHide = useCallback(() => {
    if (isDraggingRef.current) return;
    clearTimeout(hideTimeoutRef.current);
    hideTimeoutRef.current = setTimeout(() => {
      setVisible(false);
    }, 150);
  }, []);

  const cancelHide = useCallback(() => {
    clearTimeout(hideTimeoutRef.current);
  }, []);

  useEffect(() => {
    const editorDOM = editor.view.dom;

    const handleMouseMove = (event: MouseEvent) => {
      if (isDraggingRef.current) return;
      if (rafRef.current) cancelAnimationFrame(rafRef.current);

      rafRef.current = requestAnimationFrame(() => {
        if (!editor.view || editor.isDestroyed) return;

        const coords = { left: event.clientX, top: event.clientY };
        const posInfo = editor.view.posAtCoords(coords);
        if (!posInfo) {
          scheduleHide();
          return;
        }

        try {
          const $pos = editor.state.doc.resolve(posInfo.pos);
          // Get top-level block depth
          const depth = $pos.depth > 0 ? 1 : 0;
          const blockPos = $pos.before(depth);
          const node = editor.state.doc.nodeAt(blockPos);
          if (!node) {
            scheduleHide();
            return;
          }

          const dom = editor.view.nodeDOM(blockPos);
          if (!dom || !(dom instanceof HTMLElement)) {
            scheduleHide();
            return;
          }

          const rect = dom.getBoundingClientRect();
          showControls({ pos: blockPos, node, rect });
        } catch {
          scheduleHide();
        }
      });
    };

    const handleMouseLeave = () => {
      scheduleHide();
    };

    const handleDocumentDragOver = (event: DragEvent) => {
      if (!isDraggingRef.current) return;
      event.preventDefault();

      const editorRect = editor.view.dom.getBoundingClientRect();
      // Clamp coordinates to editor bounds for posAtCoords
      const clampedX = Math.max(editorRect.left + 1, Math.min(event.clientX, editorRect.right - 1));
      const clampedY = Math.max(editorRect.top + 1, Math.min(event.clientY, editorRect.bottom - 1));

      const posInfo = editor.view.posAtCoords({ left: clampedX, top: clampedY });
      if (!posInfo) return;
      try {
        const $pos = editor.state.doc.resolve(posInfo.pos);
        let blockPos: number;
        if ($pos.depth === 0) {
          // At doc level — only target last block if cursor is actually near/below it
          const lastChild = editor.state.doc.lastChild;
          if (!lastChild) return;
          blockPos = editor.state.doc.content.size - lastChild.nodeSize;
          const lastDom = editor.view.nodeDOM(blockPos);
          if (!(lastDom instanceof HTMLElement)) return;
          const lastRect = lastDom.getBoundingClientRect();
          if (clampedY < lastRect.top + lastRect.height / 2) return;
        } else {
          blockPos = $pos.before(1);
        }
        const node = editor.state.doc.nodeAt(blockPos);
        if (!node) return;
        const dom = editor.view.nodeDOM(blockPos);
        if (!(dom instanceof HTMLElement)) return;
        const rect = dom.getBoundingClientRect();
        const isBelow = $pos.depth === 0 || clampedY > rect.top + rect.height / 2;
        const gapPos = isBelow ? blockPos + node.nodeSize : blockPos;

        // Same gap as before — skip update to prevent flickering
        if (lastGapPosRef.current === gapPos) return;
        lastGapPosRef.current = gapPos;

        const newTop = isBelow ? rect.bottom : rect.top;
        lastIndicatorTopRef.current = newTop;
        setDropIndicator({
          top: newTop,
          left: editorRect.left,
          width: editorRect.width,
        });
      } catch { /* ignore */ }
    };

    const handleDocumentDragLeave = (event: DragEvent) => {
      if (!isDraggingRef.current) return;
      // Only clear indicator when leaving the window entirely
      if (!event.relatedTarget) {
        setDropIndicator(null);
      }
    };

    const handleDocumentDrop = (event: DragEvent) => {
      if (!isDraggingRef.current) return;
      const dragging = (editor.storage as Record<string, any>).blockDragHandle.dragging; // eslint-disable-line @typescript-eslint/no-explicit-any
      if (!dragging) return;

      event.preventDefault();
      (editor.storage as Record<string, any>).blockDragHandle.dragging = null; // eslint-disable-line @typescript-eslint/no-explicit-any

      const editorRect = editor.view.dom.getBoundingClientRect();
      const clampedX = Math.max(editorRect.left + 1, Math.min(event.clientX, editorRect.right - 1));
      const clampedY = Math.max(editorRect.top + 1, Math.min(event.clientY, editorRect.bottom - 1));

      const dropPos = editor.view.posAtCoords({ left: clampedX, top: clampedY });
      if (!dropPos) return;

      try {
        const $drop = editor.state.doc.resolve(dropPos.pos);
        let targetPos: number;

        if ($drop.depth === 0) {
          // Only target end of doc if cursor is actually near/below last block
          const lastChild = editor.state.doc.lastChild;
          if (!lastChild) return;
          const lastBlockPos = editor.state.doc.content.size - lastChild.nodeSize;
          const lastDom = editor.view.nodeDOM(lastBlockPos);
          if (!(lastDom instanceof HTMLElement)) return;
          const lastRect = lastDom.getBoundingClientRect();
          if (clampedY < lastRect.top + lastRect.height / 2) return;
          targetPos = editor.state.doc.content.size;
        } else {
          targetPos = $drop.before(1);
          const targetNode = editor.state.doc.nodeAt(targetPos);
          if (targetNode) {
            const targetDOM = editor.view.nodeDOM(targetPos);
            if (targetDOM instanceof HTMLElement) {
              const rect = targetDOM.getBoundingClientRect();
              if (clampedY > rect.top + rect.height / 2) {
                targetPos += targetNode.nodeSize;
              }
            }
          }
        }

        if (targetPos >= dragging.from && targetPos <= dragging.to) return;

        const node = editor.view.state.schema.nodeFromJSON(dragging.nodeJSON);
        const { tr } = editor.view.state;

        if (targetPos <= dragging.from) {
          tr.insert(targetPos, node);
          tr.delete(tr.mapping.map(dragging.from), tr.mapping.map(dragging.to));
        } else {
          tr.delete(dragging.from, dragging.to);
          tr.insert(tr.mapping.map(targetPos), node);
        }

        editor.view.dispatch(tr);
      } catch { /* ignore */ }

      isDraggingRef.current = false;
      editor.view.dom.classList.remove("block-dragging");
      setDropIndicator(null);
      lastIndicatorTopRef.current = null;
      lastGapPosRef.current = null;
      scheduleHide();
    };

    const handleGlobalDragEnd = () => {
      if (isDraggingRef.current) {
        isDraggingRef.current = false;
        editor.view.dom.classList.remove("block-dragging");
        (editor.storage as Record<string, any>).blockDragHandle.dragging = null; // eslint-disable-line @typescript-eslint/no-explicit-any
        setDropIndicator(null);
        lastIndicatorTopRef.current = null;
        lastGapPosRef.current = null;
        scheduleHide();
      }
    };

    editorDOM.addEventListener("mousemove", handleMouseMove);
    editorDOM.addEventListener("mouseleave", handleMouseLeave);
    document.addEventListener("dragover", handleDocumentDragOver);
    document.addEventListener("dragleave", handleDocumentDragLeave);
    document.addEventListener("drop", handleDocumentDrop);
    document.addEventListener("dragend", handleGlobalDragEnd);

    return () => {
      editorDOM.removeEventListener("mousemove", handleMouseMove);
      editorDOM.removeEventListener("mouseleave", handleMouseLeave);
      document.removeEventListener("dragover", handleDocumentDragOver);
      document.removeEventListener("dragleave", handleDocumentDragLeave);
      document.removeEventListener("drop", handleDocumentDrop);
      document.removeEventListener("dragend", handleGlobalDragEnd);
      clearTimeout(hideTimeoutRef.current);
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, [editor, showControls, scheduleHide]);

  const handleAdd = useCallback(() => {
    if (!activeBlock?.node) return;

    const endOfBlock = activeBlock.pos + activeBlock.node.nodeSize;

    editor
      .chain()
      .focus()
      .insertContentAt(endOfBlock, { type: "paragraph" })
      .setTextSelection(endOfBlock + 1)
      .run();

    // Wait for paragraph insertion to flush, then insert "/" + activate menu atomically
    requestAnimationFrame(() => {
      editor.commands.insertSlash();
    });
  }, [editor, activeBlock]);

  const handleDragStart = useCallback(
    (event: React.DragEvent) => {
      if (!activeBlock?.node) return;

      isDraggingRef.current = true;
      editor.view.dom.classList.add("block-dragging");
      const nodeEnd = activeBlock.pos + activeBlock.node.nodeSize;

      (editor.storage as Record<string, any>).blockDragHandle.dragging = { // eslint-disable-line @typescript-eslint/no-explicit-any
        from: activeBlock.pos,
        to: nodeEnd,
        nodeJSON: activeBlock.node.toJSON() as Record<string, unknown>,
      };

      event.dataTransfer.setData("text/plain", activeBlock.node.textContent || " ");
      event.dataTransfer.effectAllowed = "move";

      const dragGhost = document.createElement("div");
      dragGhost.style.position = "absolute";
      dragGhost.style.top = "-1000px";
      dragGhost.style.width = "20px";
      dragGhost.style.height = "20px";
      document.body.appendChild(dragGhost);
      event.dataTransfer.setDragImage(dragGhost, 0, 0);
      dragGhostRef.current = dragGhost;
    },
    [editor, activeBlock],
  );

  const handleDragEnd = useCallback(() => {
    isDraggingRef.current = false;
    editor.view.dom.classList.remove("block-dragging");
    (editor.storage as Record<string, any>).blockDragHandle.dragging = null; // eslint-disable-line @typescript-eslint/no-explicit-any
    editor.commands.focus();
    if (dragGhostRef.current) {
      dragGhostRef.current.remove();
      dragGhostRef.current = null;
    }
    setDropIndicator(null);
    lastIndicatorTopRef.current = null;
    lastGapPosRef.current = null;
    scheduleHide();
  }, [editor, scheduleHide]);

  if (!visible || !activeBlock) return null;

  const editorRect = editor.view.dom.getBoundingClientRect();

  return createPortal(
    <>
      <div
        ref={controlsRef}
        className="fixed z-50 flex flex-row items-center gap-0.5 animate-in fade-in duration-100 fill-mode-forwards rounded-md bg-background/80 backdrop-blur-sm shadow-sm border border-border/50"
        style={{
          top: activeBlock.rect.top,
          left: editorRect.left - 56,
        }}
        onMouseEnter={cancelHide}
        onMouseLeave={scheduleHide}
      >
        <button
          type="button"
          onClick={handleAdd}
          className="flex items-center justify-center h-6 w-6 rounded text-muted-foreground/50 hover:text-muted-foreground hover:bg-muted transition-colors"
          title="Add block"
        >
          <Plus size={16} />
        </button>
        <button
          type="button"
          draggable
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          className="flex items-center justify-center h-6 w-6 rounded text-muted-foreground/50 hover:text-muted-foreground hover:bg-muted transition-colors cursor-grab active:cursor-grabbing"
          title="Drag to reorder"
        >
          <GripVertical size={16} />
        </button>
      </div>
      {dropIndicator && (
        <div
          className="fixed z-50 h-0.5 bg-primary rounded-full pointer-events-none"
          style={{
            top: dropIndicator.top - 1,
            left: dropIndicator.left,
            width: dropIndicator.width,
          }}
        />
      )}
    </>,
    document.body,
  );
}
