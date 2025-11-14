package dto

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page  int `query:"page"`
	Limit int `query:"limit"`
}

// GetOffset calculates the offset from page and limit
func (p *PaginationParams) GetOffset() int {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Limit <= 0 {
		p.Limit = 100
	}
	return (p.Page - 1) * p.Limit
}

// GetLimit returns the limit, with a maximum of 1000
func (p *PaginationParams) GetLimit() int {
	if p.Limit <= 0 {
		return 100
	}
	if p.Limit > 1000 {
		return 1000
	}
	return p.Limit
}

