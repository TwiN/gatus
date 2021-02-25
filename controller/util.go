package controller

import (
	"net/http"
	"strconv"
)

func extractPageAndPageSizeFromRequest(r *http.Request) (page int, pageSize int) {
	var err error
	if pageParameter := r.URL.Query().Get("page"); len(pageParameter) == 0 {
		page = 1
	} else {
		page, err = strconv.Atoi(pageParameter)
		if err != nil {
			page = 1
		}
	}
	if pageSizeParameter := r.URL.Query().Get("pageSize"); len(pageSizeParameter) == 0 {
		pageSize = 20
	} else {
		pageSize, err = strconv.Atoi(pageSizeParameter)
		if err != nil {
			pageSize = 20
		}
		if pageSize > 100 {
			pageSize = 100
		}
	}
	return
}
