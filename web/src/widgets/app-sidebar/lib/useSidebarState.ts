import { useCallback, useEffect, useState } from "react";

const STORAGE_KEY = "task-tracker_sidebar_collapsed";

export function useSidebarState() {
  const [collapsed, setCollapsed] = useState(true);

  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved !== null) setCollapsed(saved === "true");
  }, []);

  const toggle = useCallback(() => {
    setCollapsed((prev) => {
      const next = !prev;
      localStorage.setItem(STORAGE_KEY, String(next));
      return next;
    });
  }, []);

  const setExpanded = useCallback(() => {
    setCollapsed(false);
    localStorage.setItem(STORAGE_KEY, "false");
  }, []);

  const setCollapsedState = useCallback(() => {
    setCollapsed(true);
    localStorage.setItem(STORAGE_KEY, "true");
  }, []);

  return { collapsed, toggle, setExpanded, setCollapsed: setCollapsedState };
}
