/** Returns the chosen keys with the key added, or without it once it was chosen. */
export function toggleKey(selected: string[], key: string): string[] {
  return selected.includes(key) ? selected.filter((entry) => entry !== key) : [...selected, key];
}

/**
 * Returns the chosen keys with every key from the anchor to the target in the order of the keys added,
 * the target alone without the anchor among the keys.
 */
export function selectRange(
  selected: string[],
  keys: string[],
  anchor: string | null,
  target: string,
): string[] {
  const from = anchor === null ? -1 : keys.indexOf(anchor);
  const to = keys.indexOf(target);
  if (from < 0 || to < 0) return [...new Set([...selected, target])];
  const range = keys.slice(Math.min(from, to), Math.max(from, to) + 1);
  return [...new Set([...selected, ...range])];
}
