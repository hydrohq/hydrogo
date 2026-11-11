// Pure helpers for the sidebar's grouped content-type list.
export interface GroupableCollection {
  group?: string | null;
}

export interface CollectionGroup<T extends GroupableCollection = GroupableCollection> {
  name: string;
  collections: T[];
}

export function isAdvancedGroup(name: string): boolean {
  const n = (name || '').trim().toLowerCase();
  return n === 'advanced' || n === 'gelismis' || n === 'gelişmiş';
}

export function getGroupStorageKey(groupName: string): string {
  return `hydrogo_sidebar_group_${groupName.trim().toLowerCase()}`;
}

export function getInitialGroupCollapsed(groupName: string): boolean {
  try {
    if (typeof localStorage !== 'undefined') {
      const stored = localStorage.getItem(getGroupStorageKey(groupName));
      if (stored !== null) {
        return stored === 'collapsed' || stored === 'true';
      }
    }
  } catch {}
  return isAdvancedGroup(groupName);
}

export function setGroupCollapsedStorage(groupName: string, collapsed: boolean): void {
  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(getGroupStorageKey(groupName), collapsed ? 'collapsed' : 'expanded');
    }
  } catch {}
}

/** Groups collections by their `group` (default 'Content'): regular groups keep first-appearance order, the Advanced group goes last. */
export function groupCollections<T extends GroupableCollection>(collections: T[]): CollectionGroup<T>[] {
  const groups = new Map<string, T[]>();
  for (const col of collections) {
    const name = col.group?.trim() || 'Content';
    const list = groups.get(name);
    if (list) list.push(col);
    else groups.set(name, [col]);
  }
  const all = [...groups].map(([name, cols]) => ({ name, collections: cols }));
  return [...all.filter((g) => !isAdvancedGroup(g.name)), ...all.filter((g) => isAdvancedGroup(g.name))];
}
