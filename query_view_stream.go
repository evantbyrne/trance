package trance

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

type QueryViewStream[T any] struct {
	Context     context.Context
	Error       error
	Query       *QueryStream[T]
	View        *View
	WeaveConfig WeaveConfig
}

func (stream *QueryViewStream[T]) Collect() (*QueryStream[T], *View, error) {
	return stream.Query, stream.View, stream.Error
}

func (stream *QueryViewStream[T]) Find() *MapStream {
	return stream.Select().
		Prefetch().
		Filter().
		Sort().
		Offset().
		QueryStream().
		First().
		Guard(stream.Context)
}

func (stream *QueryViewStream[T]) Filter() *QueryViewStream[T] {
	if stream.Error != nil {
		return stream
	}
	if len(stream.View.Config.Query.Filters) > 0 {
		stream.Query.Config.Filters = stream.View.Config.Query.Filters
	}
	for key, value := range stream.Request().URL.Query() {
		if keyCleaned, filtering := strings.CutPrefix(key, "filter."); filtering {
			keyParts := strings.SplitN(keyCleaned, "__", 2)
			column := keyParts[0]
			columnFilters, columnOk := stream.View.Config.AllowFilters[column]
			if !columnOk {
				stream.Error = ErrorUnauthorized{}
				return stream
			}
			_, columnWildcard := columnFilters["*"]

			operatorName := "eq"
			if len(keyParts) > 1 {
				operatorName = keyParts[1]
			}
			operator, operatorNameOk := FilterOperators[operatorName]
			if !operatorNameOk {
				stream.Error = ErrorUnauthorized{}
				return stream
			}
			if _, operatorOk := columnFilters[operatorName]; !operatorOk && !columnWildcard {
				stream.Error = ErrorUnauthorized{}
				return stream
			}

			for _, vv := range value {
				switch operator {
				case "IN", "NOT IN":
					vvs := strings.Split(vv, ",")
					vvl := make([]any, len(vvs))
					for i := range vvs {
						vvl[i] = vvs[i]
					}
					stream.Query = stream.Query.Filter(column, operator, vvl)

				case "IS", "IS NOT":
					switch vv {
					case "null":
						stream.Query = stream.Query.Filter(column, operator, nil)
					case "true":
						stream.Query = stream.Query.Filter(column, operator, true)
					case "false":
						stream.Query = stream.Query.Filter(column, operator, false)
					default:
						stream.Query = stream.Query.Filter(column, operator, vv)
					}

				default:
					stream.Query = stream.Query.Filter(column, operator, vv)
				}
			}
		}
	}
	return stream
}

func (stream *QueryViewStream[T]) Limit() *QueryViewStream[T] {
	if stream.Error != nil {
		return stream
	}
	if stream.View.Config.Query.Limit != nil {
		stream.Query = stream.Query.Limit(stream.View.Config.Query.Limit)
	}
	if limit := stream.Request().URL.Query().Get("limit"); limit != "" {
		if stream.View.Config.AllowLimit > 0 {
			if limitInt, err := strconv.Atoi(limit); err == nil && limitInt > 0 && limitInt <= stream.View.Config.AllowLimit {
				stream.Query = stream.Query.Limit(limitInt)
				return stream
			}
		}
		stream.Error = ErrorUnauthorized{}
	}
	return stream
}

func (stream *QueryViewStream[T]) List() *MapListStream {
	return stream.Select().
		Prefetch().
		Filter().
		Sort().
		Limit().
		Offset().
		QueryStream().
		All().
		Guard(stream.Context)
}

func (stream *QueryViewStream[T]) OnError(callback func(error) error) *QueryViewStream[T] {
	if stream.Error != nil {
		stream.Error = callback(stream.Error)
	}
	return stream
}

func (stream *QueryViewStream[T]) Offset() *QueryViewStream[T] {
	if stream.Error != nil {
		return stream
	}
	if stream.View.Config.Query.Offset != nil {
		stream.Query = stream.Query.Offset(stream.View.Config.Query.Offset)
	}
	if offset := stream.Request().URL.Query().Get("offset"); offset != "" {
		if stream.View.Config.AllowOffset > 0 {
			if offsetInt, err := strconv.Atoi(offset); err == nil && offsetInt > 0 && offsetInt <= stream.View.Config.AllowOffset {
				stream.Query = stream.Query.Offset(offsetInt)
				return stream
			}
		}
		stream.Error = ErrorUnauthorized{}
	}
	return stream
}

func (stream *QueryViewStream[T]) Prefetch() *QueryViewStream[T] {
	if stream.Error != nil {
		return stream
	}
	if len(stream.View.Config.AllowPrefetch) > 0 {
		_, prefetchWildcard := stream.View.Config.AllowPrefetch["*"]
		prefetch := make([]string, 0)
		for _, field := range stream.Query.Weave.Fields {
			if _, ok := stream.View.Config.AllowPrefetch[field.Name]; ok || prefetchWildcard {
				if strings.HasPrefix(field.Type.String(), "trance.OneToMany[") || strings.HasPrefix(field.Type.String(), "trance.ForeignKey[") || strings.HasPrefix(field.Type.String(), "trance.NullForeignKey[") {
					prefetch = append(prefetch, field.Name)
				}
			}
		}
		if len(prefetch) > 0 {
			stream.Query = stream.Query.FetchRelated(prefetch...)
		}
	}
	return stream
}

func (stream *QueryViewStream[T]) Request() *http.Request {
	return stream.Context.Value("Request").(*http.Request)
}

func (stream *QueryViewStream[T]) QueryStream() *QueryStream[T] {
	if stream.Error != nil {
		stream.Query.Error = stream.Error
	}
	return stream.Query
}

func (stream *QueryViewStream[T]) Select() *QueryViewStream[T] {
	if stream.Error != nil {
		return stream
	}
	columns, _ := viewFields(stream.Query.Weave, stream.View)
	if len(columns) == 0 && len(stream.Query.Config.Selected) == 0 {
		stream.Error = ErrorInternalServer{Message: "trance: No columns in view"}
	} else {
		stream.Query = stream.Query.Select(columns...)
		if stream.Query.Error != nil {
			stream.Error = stream.Query.Error
		}
	}
	return stream
}

func (stream *QueryViewStream[T]) Sort() *QueryViewStream[T] {
	if stream.Error != nil {
		return stream
	}
	if stream.Request().URL.Query().Has("sort") && len(stream.View.Config.AllowSort) > 0 {
		_, sortWildcard := stream.View.Config.AllowSort["*"]
		sortColumns := make([]string, 0)
		for _, sortColumn := range strings.Split(stream.Request().URL.Query().Get("sort"), ",") {
			s, _ := strings.CutPrefix(sortColumn, "-")
			if _, ok := stream.View.Config.AllowSort[s]; ok || sortWildcard {
				sortColumns = append(sortColumns, sortColumn)
			} else {
				stream.Error = ErrorUnauthorized{}
				return stream
			}
		}
		if len(sortColumns) > 0 {
			stream.Query = stream.Query.Sort(sortColumns...)
		}
	}
	return stream
}

func (stream *QueryViewStream[T]) Then(callback func(*QueryStream[T], *View) error) *QueryViewStream[T] {
	if stream.Error == nil {
		stream.Error = callback(stream.Query, stream.View)
	}
	return stream
}
