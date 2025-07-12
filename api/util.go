package api

import (
	"strconv"
	"github.com/gofiber/fiber/v2"
)

const (
	// DefaultPage is the default page to use if none is specified or an invalid value is provided
	DefaultPage = 1

	// DefaultPageSize is the default page size to use if none is specified or an invalid value is provided
	DefaultPageSize = 20
)

// extractPageAndPageSizeFromRequest parses `?page` and `?pageSize` from Fiber context
// and clamps them to sensible defaults, with a maximum of `maxResults` on pageSize when page==1.
func extractPageAndPageSizeFromRequest(c *fiber.Ctx, maxResults int) (page, pageSize int) {
	// Page
	if s := c.Query("page"); s == "" {
		page = DefaultPage
	} else if p, err := strconv.Atoi(s); err != nil || p < 1 {
		page = DefaultPage
	} else {
		page = p
	}

	// PageSize
	if s := c.Query("pageSize"); s == "" {
		pageSize = DefaultPageSize
	} else if sz, err := strconv.Atoi(s); err != nil || sz < 1 {
		pageSize = DefaultPageSize
	} else {
		pageSize = sz
	}

	// Enforce maximum only on first page
	if page == 1 && pageSize > maxResults {
		pageSize = maxResults
	}
	return
}
