package pagination

type PageData struct {
	Limit       int  `json:"limit"`
	Offset      int  `json:"offset"`
	TotalCount  int  `json:"totalCount"`
	HasNextPage bool `json:"hasNextPage"`
	HasPrevPage bool `json:"hasPrevPage"`
}

type Page[T any] struct {
	Data []T      `json:"data"`
	Page PageData `json:"page"`
}

func Paginate[T any](data []T, limit, offset, totalCount int) Page[T] {
	return Page[T]{
		Data: data,
		Page: PageData{
			Limit:       limit,
			Offset:      offset,
			TotalCount:  totalCount,
			HasPrevPage: offset > 0,
			HasNextPage: offset < totalCount,
		},
	}
}
