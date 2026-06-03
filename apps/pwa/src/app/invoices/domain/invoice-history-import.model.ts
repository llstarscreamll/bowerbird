const SUPPORTED_EXTENSIONS = new Set(['xml', 'pdf', 'zip']);

const SUPPORTED_MIME_TYPES = new Set(['application/xml', 'text/xml', 'application/pdf', 'application/zip', 'application/x-zip-compressed', 'multipart/x-zip']);

export const INVOICE_HISTORY_ACCEPT = '.xml,.pdf,.zip,application/xml,text/xml,application/pdf,application/zip,application/x-zip-compressed,multipart/x-zip';

export function supportsInvoiceHistoryFile(file: File): boolean {
  const extension = file.name.split('.').pop()?.toLowerCase() ?? '';

  if (SUPPORTED_EXTENSIONS.has(extension)) {
    return true;
  }

  const mimeType = file.type.toLowerCase();
  return mimeType.length > 0 && SUPPORTED_MIME_TYPES.has(mimeType);
}
