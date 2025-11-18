package paging

// SuiteStatusParams represents the parameters for suite status queries
type SuiteStatusParams struct {
	Page     int // Page number
	PageSize int // Number of results per page
}

// NewSuiteStatusParams creates a new SuiteStatusParams
func NewSuiteStatusParams() *SuiteStatusParams {
	return &SuiteStatusParams{
		Page:     1,
		PageSize: 20,
	}
}

// WithPagination sets the page and page size
func (params *SuiteStatusParams) WithPagination(page, pageSize int) *SuiteStatusParams {
	params.Page = page
	params.PageSize = pageSize
	return params
}