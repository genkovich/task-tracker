import { useCallback, useEffect, useRef, useState } from "react";

type SaveStatus = "idle" | "saving" | "saved" | "error";

interface UseAutosaveOptions {
  value: string;
  savedValue: string;
  onSave: (value: string) => Promise<void>;
  delay?: number;
}

interface UseAutosaveReturn {
  status: SaveStatus;
}

export function useAutosave({
  value,
  savedValue,
  onSave,
  delay = 1500,
}: UseAutosaveOptions): UseAutosaveReturn {
  const [status, setStatus] = useState<SaveStatus>("idle");
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingRef = useRef<string | null>(null);
  const savingRef = useRef(false);

  const save = useCallback(
    async (val: string) => {
      if (savingRef.current) return;
      savingRef.current = true;
      setStatus("saving");
      try {
        await onSave(val);
        pendingRef.current = null;
        setStatus("saved");
        setTimeout(() => setStatus("idle"), 2000);
      } catch {
        setStatus("error");
      } finally {
        savingRef.current = false;
        if (pendingRef.current !== null) {
          save(pendingRef.current);
        }
      }
    },
    [onSave],
  );

  const flushRef = useRef(save);
  flushRef.current = save;

  useEffect(() => {
    const isDirty = value !== savedValue;
    if (!isDirty) return;

    pendingRef.current = value;

    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      save(value);
    }, delay);

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [value, savedValue, delay, save]);

  // Flush on unmount
  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      if (pendingRef.current !== null) {
        flushRef.current(pendingRef.current);
      }
    };
  }, []);

  // Cmd+S / Ctrl+S immediate save
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "s") {
        e.preventDefault();
        if (value !== savedValue) {
          if (timerRef.current) clearTimeout(timerRef.current);
          save(value);
        }
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [value, savedValue, save]);

  return { status };
}
