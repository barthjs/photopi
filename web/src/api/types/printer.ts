export type PrintCancelRequest = {
  printerName: string;
};

export type PrintResumeRequest = {
  printerName: string;
};

export type PrintTestRequest = {
  printerName: string;
};

export type PrinterResponse = {
  name: string;
  status: string;
};
