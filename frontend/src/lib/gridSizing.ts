/**
 * Column width for the matrix grid: stretches to fill the available width within
 * [min, max], falling back to `min` (and letting the caller's scroll container take
 * over) when even the minimum doesn't fit all buckets.
 */
export function computeCellWidth(
  containerWidth: number,
  reservedWidth: number,
  bucketCount: number,
  min: number,
  max: number
): number {
  if (bucketCount === 0) return min
  const columnsAreaWidth = Math.max(0, containerWidth - reservedWidth)
  if (bucketCount * min > columnsAreaWidth) return min
  return Math.min(max, columnsAreaWidth / bucketCount)
}
