package server

type PaginationFilters struct {
	Page     int32 `query:"page" validate:"min=1,max=10000"`
	PageSize int32 `query:"pageSize" validate:"min=1,max=100"`
}

func (f PaginationFilters) limit() int32 {
	return f.PageSize
}

func (f PaginationFilters) offset() int32 {
	return (f.Page - 1) * f.PageSize
}

type Pagination struct {
	Page        int  `json:"page"`
	PageSize    int  `json:"pageSize"`
	TotalItems  int  `json:"totalItems"`
	TotalPages  int  `json:"totalPages"`
	HasNext     bool `json:"hasNext"`
	HasPrevious bool `json:"hasPrevious"`
}

func calculatePagination(totalItems, page, pageSize int) Pagination {
	if totalItems == 0 {
		return Pagination{Page: page, PageSize: pageSize}
	}

	return Pagination{
		Page:        page,
		PageSize:    pageSize,
		TotalItems:  totalItems,
		TotalPages:  (totalItems + pageSize - 1) / pageSize,
		HasNext:     page*pageSize < totalItems,
		HasPrevious: page > 1,
	}
}
