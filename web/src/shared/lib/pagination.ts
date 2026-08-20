interface PageResult<T> {
  items: T[];
  has_next: boolean;
}

export async function fetchAllPages<T>(
  fetchPage: (after?: string) => Promise<PageResult<T>>,
  getCursor: (item: T) => string,
): Promise<T[]> {
  const all: T[] = [];
  let after: string | undefined;
  for (;;) {
    const page = await fetchPage(after);
    all.push(...page.items);
    if (!page.has_next || page.items.length === 0) break;
    after = getCursor(page.items[page.items.length - 1]);
  }
  return all;
}