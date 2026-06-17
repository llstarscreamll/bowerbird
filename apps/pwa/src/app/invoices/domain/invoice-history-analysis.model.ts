export interface InvoiceHistoryAnalyzeFileReference {
  name: string;
  type: string;
  url: string;
}

export interface QueueInvoiceExtractionFile {
  name: string;
  path: string;
  mime_type: string;
}

export interface StartInvoiceHistoryAnalysisRequest {
  data: {
    id: string;
    type: 'queue-invoice-extraction';
    attributes: {
      files: QueueInvoiceExtractionFile[];
    };
  };
}
