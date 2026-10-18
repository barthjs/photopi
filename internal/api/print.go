package api

import (
	"net/http"
)

// printerListHandler lists every CUPS printer queue known to the system.
//
//	@Summary		List printers
//	@Description	Lists every CUPS printer queue known to the system (Admin only).
//	@Tags			print
//	@Produce		json
//	@Success		200	{array}		PrinterResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/print/printers [get]
func (a *API) printerListHandler(w http.ResponseWriter, r *http.Request) {
	printers, err := a.printService.ListPrinters(r.Context())
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	response := make([]PrinterResponse, 0, len(printers))
	for _, p := range printers {
		response = append(response, PrinterResponse{
			Name:   p.Name,
			Status: string(p.Status),
		})
	}

	a.writeJSON(w, r, http.StatusOK, response, nil)
}

// PrinterResponse is a CUPS printer queue with its status.
type PrinterResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PrintTestRequest names the printer queue to check.
type PrintTestRequest struct {
	PrinterName string `json:"printerName" validate:"required"`
}

// printTestHandler verifies that the submitted printer queue exists and is not disabled,
// without submitting anything to print, so testing a printer never wastes paper.
//
//	@Summary		Test printer
//	@Description	Verifies that the submitted CUPS printer queue exists and is not disabled, without printing anything (Admin only).
//	@Tags			print
//	@Accept			json
//	@Produce		json
//	@Param			request	body		PrintTestRequest	true	"Printer to test"
//	@Success		200		{object}	ConfigTestResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Security		SessionAuth
//	@Router			/print/test [post]
func (a *API) printTestHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[PrintTestRequest](w, r)
	if !ok {
		return
	}

	ok, message := a.printService.TestConnection(r.Context(), req.PrinterName)
	a.writeJSON(w, r, http.StatusOK, ConfigTestResponse{
		OK:      ok,
		Message: message,
	}, nil)
}

// PrintResumeRequest names the printer queue to resume.
type PrintResumeRequest struct {
	PrinterName string `json:"printerName" validate:"required"`
}

// printResumeHandler enables a printer queue again and releases its held jobs.
//
//	@Summary		Resume printer
//	@Description	Enables a CUPS printer queue again and releases its held jobs, for example after the printer stopped it because of an error (Admin only).
//	@Tags			print
//	@Accept			json
//	@Param			request	body	PrintResumeRequest	true	"Printer to resume"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		422	{object}	ValidationErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/print/resume [post]
func (a *API) printResumeHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[PrintResumeRequest](w, r)
	if !ok {
		return
	}

	if err := a.printService.Resume(r.Context(), req.PrinterName); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PrintCancelRequest names the printer queue whose jobs to cancel.
type PrintCancelRequest struct {
	PrinterName string `json:"printerName" validate:"required"`
}

// printCancelHandler cancels every job of a printer queue.
//
//	@Summary		Cancel print jobs
//	@Description	Cancels every job of a CUPS printer queue (Admin only).
//	@Tags			print
//	@Accept			json
//	@Param			request	body	PrintCancelRequest	true	"Printer whose jobs to cancel"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		422	{object}	ValidationErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/print/cancel [post]
func (a *API) printCancelHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[PrintCancelRequest](w, r)
	if !ok {
		return
	}

	if err := a.printService.Cancel(r.Context(), req.PrinterName); err != nil {
		a.respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
