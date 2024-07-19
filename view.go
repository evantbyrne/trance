package trance

type ViewConfig struct {
	AllowFields   map[string]bool
	AllowFilters  map[string]map[string]bool
	AllowLimit    int
	AllowOffset   int
	AllowPrefetch map[string]bool
	AllowSort     map[string]bool
	Query         QueryConfig
}

type View struct {
	Config ViewConfig
}

func (view *View) AllowFields(fields ...string) *View {
	for _, field := range fields {
		view.Config.AllowFields[field] = true
	}
	return view
}

func (view *View) AllowFilter(column string, operators ...string) *View {
	if _, ok := view.Config.AllowFilters[column]; !ok {
		view.Config.AllowFilters[column] = make(map[string]bool, 0)
	}
	if len(operators) == 0 {
		view.Config.AllowFilters[column]["eq"] = true
	} else {
		for _, operator := range operators {
			if _, ok := FilterOperators[operator]; ok || operator == "*" {
				view.Config.AllowFilters[column][operator] = true
			}
		}
	}
	return view
}

func (view *View) AllowLimit(limit int) *View {
	view.Config.AllowLimit = limit
	return view
}

func (view *View) AllowOffset(offset int) *View {
	view.Config.AllowOffset = offset
	return view
}

func (view *View) AllowPrefetch(fields ...string) *View {
	for _, field := range fields {
		view.Config.AllowPrefetch[field] = true
	}
	return view
}

func (view *View) AllowSort(columns ...string) *View {
	for _, column := range columns {
		view.Config.AllowSort[column] = true
	}
	return view
}

func (view *View) Filter(left any, operator string, right any) *View {
	if len(view.Config.Query.Filters) > 0 {
		view.Config.Query.Filters = append(view.Config.Query.Filters, FilterClause{
			Rule: "AND",
		})
	}
	view.Config.Query.Filters = append(view.Config.Query.Filters, Q(left, operator, right))
	return view
}

func (view *View) Limit(limit int) *View {
	view.Config.Query.Limit = limit
	return view
}

func (view *View) Offset(offset int) *View {
	view.Config.Query.Offset = offset
	return view
}

func AllowFields(fields ...string) *View {
	return Deny().AllowFields(fields...)
}

func AllowFilter(column string, operators ...string) *View {
	return Deny().AllowFilter(column, operators...)
}

func AllowLimit(limit int) *View {
	return Deny().AllowLimit(limit)
}

func AllowOffset(offset int) *View {
	return Deny().AllowOffset(offset)
}

func AllowPrefetch(fields ...string) *View {
	return Deny().AllowPrefetch(fields...)
}

func AllowSort(columns ...string) *View {
	return Deny().AllowSort(columns...)
}

func Deny() *View {
	return &View{
		Config: ViewConfig{
			AllowFields:   make(map[string]bool, 0),
			AllowFilters:  make(map[string]map[string]bool, 0),
			AllowPrefetch: make(map[string]bool, 0),
			AllowSort:     make(map[string]bool, 0),
		},
	}
}
