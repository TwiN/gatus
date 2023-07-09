package api

import (
	"strconv"

	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/gofiber/fiber/v2"
)

const (
	// DefaultPage is the default page to use if none is specified or an invalid value is provided
	DefaultPage = 1

	// DefaultPageSize is the default page siZE to use if none is specified or an invalid value is provided
	DefaultPageSize = 20

	// MaximumPageSize is the maximum page size allowed
	MaximumPageSize = common.MaximumNumberOfResults
)

func extractPageAndPageSizeFromRequest(c *fiber.Ctx) (page, pageSize int) {
	var err error
	if pageParameter := c.Query("page"); len(pageParameter) == 0 {
		page = DefaultPage
	} else {
		page, err = strconv.Atoi(pageParameter)
		if err != nil {
			page = DefaultPage
		}
		if page < 1 {
			page = DefaultPage
		}
	}
	if pageSizeParameter := c.Query("pageSize"); len(pageSizeParameter) == 0 {
		pageSize = DefaultPageSize
	} else {
		pageSize, err = strconv.Atoi(pageSizeParameter)
		if err != nil {
			pageSize = DefaultPageSize
		}
		if pageSize > MaximumPageSize {
			pageSize = MaximumPageSize
		} else if pageSize < 1 {
			pageSize = DefaultPageSize
		}
	}
	return
}
