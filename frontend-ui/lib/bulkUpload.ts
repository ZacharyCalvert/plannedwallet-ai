export const BULK_UPLOAD_BATCH_SIZE = 1000;

export function chunkItemsForBulkUpload<T>(
  items: T[],
  batchSize: number = BULK_UPLOAD_BATCH_SIZE
): T[][] {
  if (batchSize <= 0) {
    throw new Error('batchSize must be positive');
  }
  const chunks: T[][] = [];
  for (let i = 0; i < items.length; i += batchSize) {
    chunks.push(items.slice(i, i + batchSize));
  }
  return chunks;
}
