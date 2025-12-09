package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPagination(t *testing.T) {
	tests := []struct {
		name               string
		totalItems         int
		page               int
		pageSize           int
		expectedPagination Pagination
	}{
		{
			name:               "page 1, page size 10",
			totalItems:         100,
			page:               1,
			pageSize:           10,
			expectedPagination: Pagination{Page: 1, PageSize: 10, TotalPages: 10, TotalItems: 100, HasNext: true, HasPrevious: false},
		},
		{
			name:               "page 5, page size 10",
			totalItems:         100,
			page:               5,
			pageSize:           10,
			expectedPagination: Pagination{Page: 5, PageSize: 10, TotalPages: 10, TotalItems: 100, HasNext: true, HasPrevious: true},
		},
		{
			name:               "page 1, page size 100",
			totalItems:         100,
			page:               1,
			pageSize:           100,
			expectedPagination: Pagination{Page: 1, PageSize: 100, TotalPages: 1, TotalItems: 100},
		},
		{
			name:               "page 2, page size 100",
			totalItems:         100,
			page:               2,
			pageSize:           100,
			expectedPagination: Pagination{Page: 2, PageSize: 100, TotalPages: 1, TotalItems: 100, HasPrevious: true},
		},
		{
			name:               "page 1, page size 1",
			totalItems:         100,
			page:               1,
			pageSize:           1,
			expectedPagination: Pagination{Page: 1, PageSize: 1, TotalPages: 100, TotalItems: 100, HasNext: true, HasPrevious: false},
		},
		{
			name:               "page 2, page size 1",
			totalItems:         100,
			page:               2,
			pageSize:           1,
			expectedPagination: Pagination{Page: 2, PageSize: 1, TotalPages: 100, TotalItems: 100, HasNext: true, HasPrevious: true},
		},
		{
			name:               "total items 0",
			totalItems:         0,
			page:               1,
			pageSize:           10,
			expectedPagination: Pagination{Page: 1, PageSize: 10, TotalPages: 0, TotalItems: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPagination := calculatePagination(tt.totalItems, tt.page, tt.pageSize)
			assert.Equal(t, tt.expectedPagination, actualPagination)
		})
	}
}
