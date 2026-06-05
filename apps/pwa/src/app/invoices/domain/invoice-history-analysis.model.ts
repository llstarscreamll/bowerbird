export interface InvoiceHistoryAnalyzeFileReference {
  name: string;
  type: string;
  url: string;
}

export interface StartInvoiceHistoryAnalysisRequest {
  files: InvoiceHistoryAnalyzeFileReference[];
}
