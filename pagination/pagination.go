package pagination

import (
	"errors"

	"bio/specs"

	"github.com/vagruchi/sqb"
)

// nolint:deadcode,varcheck // it's need
var keysToOrderByCend = map[string]bool{
	"asc":  false,
	"desc": true,
}

type PaginationPolitics struct {
	MaxLimit       int
	DefaultLimit   int
	OrderByMappgin map[string]string
	DefaultOrder   *OrderBy
}

// Pagination is a struct for guides Pagination
type Pagination struct {
	Limit, Offset, Total int
	OrderBy              []*OrderBy
	DefaultOrder         *OrderBy
}

type OrderBy struct {
	Sortname string
	Desc     bool
}

func NewOrderByFields(sortname string, desc bool) *OrderBy {
	return &OrderBy{
		Sortname: sortname,
		Desc:     desc,
	}
}

func NewPagination(limit, offset int, skipCount bool,
	orderBy ...*OrderBy) *Pagination {
	return &Pagination{
		Limit:   limit,
		Offset:  offset,
		OrderBy: orderBy,
	}
}

func (pp PaginationPolitics) MakePagination(pgn *specs.Pagination, sort *specs.Sort) (Pagination, error) {
	if pgn == nil {
		pgn = &specs.Pagination{
			Offset: 0,
			Limit:  pp.DefaultLimit,
		}
	}

	page := Pagination{}

	page.Limit = pgn.Limit

	if page.Limit == 0 {
		page.Limit = pp.DefaultLimit
	}

	if page.Limit == 0 || page.Limit > pp.MaxLimit {
		page.Limit = pp.MaxLimit
	}

	page.Offset = pgn.Offset
	if pgn.Offset < 0 {
		page.Offset = 0
	}

	var sortPath = pp.OrderByMappgin

	if sort != nil {
		if sort.SortKey == "" || sort.SortOrder == "" {
			page.DefaultOrder = pp.DefaultOrder
			return page, nil
		}

		var isDesc, ok bool
		var column string

		isDesc, ok = keysToOrderByCend[sort.SortOrder]
		if !ok {
			return Pagination{}, errors.New("invalid argument: sort.sortKey")
		}

		column, ok = sortPath[sort.SortKey]
		if !ok {
			return Pagination{}, errors.New("invalid argument: sort.SortOrder")
		}

		page.OrderBy = append(page.OrderBy, NewOrderByFields(column, isDesc))
	}

	if len(page.OrderBy) == 0 {
		page.DefaultOrder = pp.DefaultOrder
	}

	return page, nil
}

// case insensitive
func (p *Pagination) AddOrderByAsc(field string) {
	p.OrderBy = append(p.OrderBy, NewOrderByFields(field, false))
}

// case insensitive
func (p *Pagination) AddOrderByDesc(field string) {
	p.OrderBy = append(p.OrderBy, NewOrderByFields(field, true))
}

func (p *Pagination) Apply(q *sqb.SelectStmt) *sqb.SelectStmt {
	stmt := *q

	if p == nil {
		return &stmt
	}

	orderByStmt := []sqb.OrderByElem{}

	for _, orderBy := range p.OrderBy {
		ob := sqb.Asc(sqb.Column(orderBy.Sortname))

		if orderBy.Desc {
			ob = sqb.Desc(sqb.Column(orderBy.Sortname))
		}

		orderByStmt = append(orderByStmt, ob)
	}

	stmt = stmt.OrderBy(orderByStmt...)

	if p.Limit > 0 {
		stmt = stmt.Limit(uint64(p.Limit))

		if p.Offset > 0 {
			stmt = stmt.Offset(uint64(p.Offset))
		}
	}

	return &stmt
}
