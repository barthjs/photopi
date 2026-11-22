package api

type BulkResponse struct {
	Succeeded int      `json:"succeeded"`
	Failed    int      `json:"failed"`
	Skipped   int      `json:"skipped"`
	FailedIDs []string `json:"failedIds"`
}

// bulkIDs holds the IDs a bulk action applies to.
type bulkIDs struct {
	IDs []string `json:"ids" validate:"required,min=1,max=200,unique,dive,required"`
}
