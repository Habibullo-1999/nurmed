package structs

type Pagination struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

func (p *Pagination) Validate() {
	if p.Offset < 0 {
		p.Offset = 0
	}

	if p.Limit <= 0 {
		p.Limit = 20
	}

	if p.Limit > 200 {
		p.Limit = 200
	}
}
