package trance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

var db_ *sql.DB

var FilterOperators = map[string]string{
	"eq":     "=",
	"in":     "IN",
	"is":     "IS",
	"is_not": "IS NOT",
	"lt":     "<",
	"lte":    "<=",
	"not":    "!=",
	"not_in": "NOT IN",
	"gt":     ">",
	"gte":    ">=",
}

type Actioner[T any] interface {
	Action(http.ResponseWriter, *http.Request, *sql.DB, *Weave[T], *T) (*T, error)
}

type FormRenderer[T Viewer] interface {
	RenderError(http.ResponseWriter, *http.Request, error)
	RenderFind(http.ResponseWriter, *http.Request, *Weave[T], *View, *T)
	RenderFormErrors(http.ResponseWriter, *http.Request, error)
	RenderList(http.ResponseWriter, *http.Request, *Weave[T], *View, []*T)
}

type FormWithAfter[T any] interface {
	After(*http.Request, *T) (*T, error)
}

type FormWithBefore[T any] interface {
	Before(*http.Request, *T) (*T, error)
}

type GuardedFormViewer interface {
	GuardForm(*http.Request, *View, any) (*View, error)
}

type GuardedViewer interface {
	GuardSelect(context.Context, *View) (*View, error)
}

type ResponseRenderer[T Viewer] interface {
	RenderError(http.ResponseWriter, *http.Request, error)
	RenderFind(http.ResponseWriter, *http.Request, *Weave[T], *View, *T)
	RenderList(http.ResponseWriter, *http.Request, *Weave[T], *View, []*T)
}

type Viewer interface {
	ViewSelect(context.Context) *View
}

func AllJson[C any, T Viewer, FormCreate any, FormDelete any, FormEdit any]() func(*Strand) error {
	handleCreate := CreateJson[T, FormCreate]()
	handleDelete := DeleteJson[T, FormDelete]()
	handleEdit := EditJson[T, FormEdit]()
	handleFind := FindJson[T]()
	handleList := ListJson[T]()

	return unwrap(func(s *Strand) error {
		r := s.Request()
		_, r.URL.Path = shiftPath(r.URL.Path)
		switch r.URL.Path {
		case "/create.json":
			handleCreate(s)
			return nil
		case "/delete.json":
			handleDelete(s)
			return nil
		case "/edit.json":
			handleEdit(s)
			return nil
		case "/find.json":
			handleFind(s)
			return nil
		case "/list.json":
			handleList(s)
			return nil
		default:
			return ErrorNotFound{}
		}
	})
}

func Create[T Viewer, F any](renderer FormRenderer[T]) func(*Strand) error {
	if Database() == nil {
		panic("trance: Database not registered. Please call 'trance.UseDatabase(*sql.DB)' before 'trance.Create[T Viewer, F any](FormRenderer[T])'")
	}

	form := Use[F]()
	weave := Use[T]()

	for column, field := range form.Fields {
		if _, ok := weave.Fields[column]; !ok {
			panic(fmt.Sprintf("trance: Form field '%s' mapped to '%s' does not exist on model '%s'", field.Name, column, weave.Type.Name()))
		}
	}

	// Insert.
	return func(s *Strand) error {
		if s.Request().Method != "POST" {
			return ErrorMethodNotAllowed{AllowedMethod: "POST"}
		}
		var temp T
		view := temp.ViewSelect(s.Context)

		// Validate Form.
		modelForm, err := form.Validate(s.Request())
		if err != nil {
			renderer.RenderFormErrors(s.Response, s.Request(), err)
			return nil
		}

		// Guard Form.
		if gfv, ok := any(temp).(GuardedFormViewer); ok {
			var err error
			view, err = gfv.GuardForm(s.Request(), view, modelForm)
			if err != nil {
				renderer.RenderError(s.Response, s.Request(), err)
				return nil
			}
		}

		// Convert to model.
		data, err := form.ToMap(modelForm)
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}
		record, err := weave.ScanMap(data)
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}

		// Before.
		if mfv, ok := any(modelForm).(FormWithBefore[T]); ok {
			var err error
			record, err = mfv.Before(s.Request(), record)
			if err != nil {
				renderer.RenderError(s.Response, s.Request(), err)
				return nil
			}
		}

		results, _, err := Query[T]().Insert(record).Collect()
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}
		id, _ := results.LastInsertId()
		record, err = Query[T]().Filter(weave.PrimaryColumn, "=", id).First().Collect()
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}

		// After.
		if mfv, ok := any(modelForm).(FormWithAfter[T]); ok {
			record, err = mfv.After(s.Request(), record)
			if err != nil {
				renderer.RenderError(s.Response, s.Request(), err)
				return nil
			}
		}

		// Render.
		renderer.RenderFind(s.Response, s.Request(), weave, view, record)
		return nil
	}
}

func Database() *sql.DB {
	return db_
}

func Delete[T Viewer, F any](renderer FormRenderer[T]) func(*Strand) error {
	if Database() == nil {
		panic("trance: Database not registered. Please call 'trance.UseDatabase(*sql.DB)' before 'trance.Delete[T Viewer, F any](ResponseRenderer[T])'")
	}

	form := Use[F]()
	weave := Use[T]()

	for column, field := range form.Fields {
		if _, ok := weave.Fields[column]; !ok {
			panic(fmt.Sprintf("trance: Form field '%s' mapped to '%s' does not exist on weave '%s'", field.Name, column, weave.Type.Name()))
		}
	}

	return func(s *Strand) error {
		if s.Request().Method != "POST" {
			return ErrorMethodNotAllowed{AllowedMethod: "POST"}
		}
		var record *T
		var temp T
		view := temp.ViewSelect(s.Context)

		// Find.
		record, err := findWithView(s.Request(), weave, view)
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}

		// Validate Form.
		modelForm, err := form.Validate(s.Request())
		if err != nil {
			renderer.RenderFormErrors(s.Response, s.Request(), err)
			return nil
		}

		// Guard Form.
		if gfv, ok := any(record).(GuardedFormViewer); ok {
			var err error
			view, err = gfv.GuardForm(s.Request(), view, modelForm)
			if err != nil {
				renderer.RenderError(s.Response, s.Request(), err)
				return nil
			}
		}

		// Before.
		if mfv, ok := any(modelForm).(FormWithBefore[T]); ok {
			var err error
			record, err = mfv.Before(s.Request(), record)
			if err != nil {
				renderer.RenderError(s.Response, s.Request(), err)
				return nil
			}
		}

		// Delete.
		value := reflect.ValueOf(record).Elem()
		id := value.FieldByName(weave.PrimaryField).Interface()
		err = Query[T]().Filter(weave.PrimaryColumn, "=", id).Delete().Error
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}

		// After.
		if mfv, ok := any(modelForm).(FormWithAfter[T]); ok {
			record, err = mfv.After(s.Request(), record)
			if err != nil {
				renderer.RenderError(s.Response, s.Request(), err)
				return nil
			}
		}

		// Render.
		renderer.RenderFind(s.Response, s.Request(), weave, view, record)
		return nil
	}
}

func Edit[T Viewer, F any](renderer FormRenderer[T]) func(*Strand) error {
	if Database() == nil {
		panic("trance: Database not registered. Please call 'trance.UseDatabase(*sql.DB)' before 'trance.Edit[T Viewer, F any](FormRenderer[T])'")
	}

	form := Use[F]()
	weave := Use[T]()

	for column, field := range form.Fields {
		if _, ok := weave.Fields[column]; !ok {
			panic(fmt.Sprintf("trance: Form field '%s' mapped to '%s' does not exist on model '%s'", field.Name, column, weave.Type.Name()))
		}
	}

	// Edit.
	return func(s *Strand) error {
		if s.Request().Method != "POST" {
			return ErrorMethodNotAllowed{AllowedMethod: "POST"}
		}
		var temp T
		view := temp.ViewSelect(s.Context)

		// Find.
		record, err := findWithView(s.Request(), weave, view)
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}

		// Validate Form.
		modelForm, err := form.Validate(s.Request())
		if err != nil {
			renderer.RenderFormErrors(s.Response, s.Request(), err)
			return nil
		}

		// Guard Form.
		if gfv, ok := any(record).(GuardedFormViewer); ok {
			var err error
			view, err = gfv.GuardForm(s.Request(), view, modelForm)
			if err != nil {
				renderer.RenderError(s.Response, s.Request(), err)
				return nil
			}
		}

		// Merge form and record.
		data, err := weave.ToMap(record)
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}
		dataForm, err := form.ToMap(modelForm)
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}
		maps.Copy(data, dataForm)
		record, err = weave.ScanMap(data)
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}

		// Before.
		if mfv, ok := any(modelForm).(FormWithBefore[T]); ok {
			var err error
			record, err = mfv.Before(s.Request(), record)
			if err != nil {
				renderer.RenderError(s.Response, s.Request(), err)
				return nil
			}
		}

		// Convert to map.
		// TODO: Diff.
		data, err = weave.ToMap(record)
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}

		// Edit.
		value := reflect.ValueOf(record).Elem()
		id := value.FieldByName(weave.PrimaryField).Interface()
		err = Query[T]().Filter(weave.PrimaryColumn, "=", id).UpdateMap(data).Error
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}
		record, err = Query[T]().Filter(weave.PrimaryColumn, "=", id).First().Collect()
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}

		// After.
		if mfv, ok := any(modelForm).(FormWithAfter[T]); ok {
			record, err = mfv.After(s.Request(), record)
			if err != nil {
				renderer.RenderError(s.Response, s.Request(), err)
				return nil
			}
		}

		// Render.
		renderer.RenderFind(s.Response, s.Request(), weave, view, record)
		return nil
	}
}

func fieldViewers[T any](weave *Weave[T]) map[string]Viewer {
	record := weave.Zero()
	value := reflect.ValueOf(&record).Elem()
	guards := make(map[string]Viewer, 0)
	for _, field := range weave.Fields {
		if strings.HasPrefix(field.Type.String(), "trance.ForeignKey[") || strings.HasPrefix(field.Type.String(), "trance.NullForeignKey[") || strings.HasPrefix(field.Type.String(), "trance.OneToMany[") {
			valueField := reflect.Indirect(value.FieldByName(field.Name))
			qWeave := valueField.Addr().MethodByName("Weave").Call(nil)
			qZero := reflect.Indirect(qWeave[0]).Addr().MethodByName("Zero").Call(nil)
			if rowViewType, ok := reflect.Indirect(qZero[0]).Interface().(Viewer); ok {
				guards[field.Name] = rowViewType
			}
		}
	}
	return guards
}

func Find[T Viewer](renderer ResponseRenderer[T]) func(*Strand) error {
	if Database() == nil {
		panic("trance: Database not registered. Please call 'trance.UseDatabase(*sql.DB)' before 'trance.Find[T Viewer](ResponseRenderer[T])'")
	}

	return func(s *Strand) error {
		if s.Request().Method != "GET" {
			return ErrorMethodNotAllowed{AllowedMethod: "GET"}
		}
		var temp T
		weave := Use[T]()
		view := temp.ViewSelect(s.Context)

		// Find.
		record, err := findWithView(s.Request(), weave, view)
		if err != nil {
			renderer.RenderError(s.Response, s.Request(), err)
			return nil
		}

		// Render.
		renderer.RenderFind(s.Response, s.Request(), weave, view, record)
		return nil
	}
}

func findWithView[T Viewer](r *http.Request, weave *Weave[T], view *View) (*T, error) {
	columns, _ := viewFields(weave, view)

	if len(columns) == 0 {
		return nil, ErrorInternalServer{Message: "trance: No columns in view"}
	}
	query := Query[T]().Select(columns...)
	query = queryPrefetcher(r, query, view)
	query = queryFilter(r, query, view)
	query = querySorter(r, query, view)
	query = queryLimiter(r, query, view)
	query = queryOffsetter(r, query, view)

	if query.Error != nil {
		return nil, query.Error
	}

	if len(query.Config.Filters) == 0 {
		return nil, ErrorBadRequest{Message: "No filters provided"}
	}

	record, err := query.First().Collect()
	if err != nil {
		return nil, err
	}
	return record, nil
}

func Guard[T any](ctx context.Context, record *T, configs ...WeaveConfig) (map[string]any, error) {
	var temp T
	if trv, ok := any(temp).(Viewer); ok {
		view := trv.ViewSelect(ctx)

		// Guard Select.
		if gfv, ok := any(record).(GuardedViewer); ok {
			var err error
			view, err = gfv.GuardSelect(ctx, view)
			if err != nil {
				return nil, err
			}
		}

		var weave *Weave[T]
		if len(configs) > 0 {
			weave = UseWith[T](configs[0])
		} else {
			weave = Use[T]()
		}
		_, fieldsLower := viewFields(weave, view)

		// Generate a map of fields to views.
		fieldViews := requestFieldViews(ctx, fieldViewers(weave))

		return recordGuard[T](fieldsLower, fieldViews, weave.ToJsonMap(record)), nil
	}

	return nil, fmt.Errorf("trance: '%T' does not implement 'trance.Viewer'", record)
}

func GuardList[T any](ctx context.Context, records []*T, configs ...WeaveConfig) ([]map[string]any, error) {
	var temp T
	if trv, ok := any(temp).(Viewer); ok {
		defaultView := trv.ViewSelect(ctx)

		results := make([]map[string]any, len(records))
		for i, record := range records {
			view := defaultView

			// Guard Select.
			if gfv, ok := any(record).(GuardedViewer); ok {
				var err error
				view, err = gfv.GuardSelect(ctx, view)
				if err != nil {
					return nil, err
				}
			}

			var weave *Weave[T]
			if len(configs) > 0 {
				weave = UseWith[T](configs[0])
			} else {
				weave = Use[T]()
			}
			_, fieldsLower := viewFields(weave, view)

			// Generate a map of fields to views.
			fieldViews := requestFieldViews(ctx, fieldViewers(weave))

			results[i] = recordGuard[T](fieldsLower, fieldViews, weave.ToJsonMap(record))
		}

		return results, nil
	}

	return nil, fmt.Errorf("trance: '%T' does not implement 'trance.Viewer'", temp)
}

func List[T Viewer](renderer ResponseRenderer[T]) func(*Strand) error {
	if Database() == nil {
		panic("trance: Database not registered. Please call 'trance.UseDatabase(*sql.DB)' before 'trance.List[T Viewer](ResponseRenderer[T])'")
	}

	// List.
	return func(s *Strand) error {
		if s.Request().Method != "GET" {
			return ErrorMethodNotAllowed{AllowedMethod: "GET"}
		}
		var records []*T
		var temp T
		weave := Use[T]()
		view := temp.ViewSelect(s.Context)
		columns, _ := viewFields(weave, view)

		if len(columns) > 0 {
			query := Query[T]().Select(columns...)
			query = queryPrefetcher(s.Request(), query, view)
			query = queryFilter(s.Request(), query, view)
			query = querySorter(s.Request(), query, view)
			query = queryLimiter(s.Request(), query, view)
			query = queryOffsetter(s.Request(), query, view)

			if query.Error != nil {
				renderer.RenderError(s.Response, s.Request(), query.Error)
				return nil
			}

			var err error
			records, err = query.All().Collect()
			if err != nil {
				renderer.RenderError(s.Response, s.Request(), err)
				return nil
			}
		}

		// Render.
		renderer.RenderList(s.Response, s.Request(), weave, view, records)
		return nil
	}
}

func queryPrefetcher[T Viewer](_ *http.Request, query *QueryStream[T], view *View) *QueryStream[T] {
	if len(view.Config.AllowPrefetch) > 0 {
		_, prefetchWildcard := view.Config.AllowPrefetch["*"]
		prefetch := make([]string, 0)
		for _, field := range query.Weave.Fields {
			if _, ok := view.Config.AllowPrefetch[field.Name]; ok || prefetchWildcard {
				if strings.HasPrefix(field.Type.String(), "trance.OneToMany[") || strings.HasPrefix(field.Type.String(), "trance.ForeignKey[") || strings.HasPrefix(field.Type.String(), "trance.NullForeignKey[") {
					prefetch = append(prefetch, field.Name)
				}
			}
		}
		if len(prefetch) > 0 {
			query = query.FetchRelated(prefetch...)
		}
	}
	return query
}

func queryFilter[T Viewer](r *http.Request, query *QueryStream[T], view *View) *QueryStream[T] {
	if len(view.Config.Query.Filters) > 0 {
		query.Config.Filters = view.Config.Query.Filters
	}
	for key, value := range r.URL.Query() {
		if keyCleaned, filtering := strings.CutPrefix(key, "filter."); filtering {
			keyParts := strings.SplitN(keyCleaned, "__", 2)
			column := keyParts[0]
			columnFilters, columnOk := view.Config.AllowFilters[column]
			if !columnOk {
				query.Error = ErrorUnauthorized{}
				return query
			}
			_, columnWildcard := columnFilters["*"]

			operatorName := "eq"
			if len(keyParts) > 1 {
				operatorName = keyParts[1]
			}
			operator, operatorNameOk := FilterOperators[operatorName]
			if !operatorNameOk {
				query.Error = ErrorUnauthorized{}
				return query
			}
			if _, operatorOk := columnFilters[operatorName]; !operatorOk && !columnWildcard {
				query.Error = ErrorUnauthorized{}
				return query
			}

			for _, vv := range value {
				switch operator {
				case "IN", "NOT IN":
					vvs := strings.Split(vv, ",")
					vvl := make([]any, len(vvs))
					for i := range vvs {
						vvl[i] = vvs[i]
					}
					query = query.Filter(column, operator, vvl)

				case "IS", "IS NOT":
					switch vv {
					case "null":
						query = query.Filter(column, operator, nil)
					case "true":
						query = query.Filter(column, operator, true)
					case "false":
						query = query.Filter(column, operator, false)
					default:
						query = query.Filter(column, operator, vv)
					}

				default:
					query = query.Filter(column, operator, vv)
				}
			}
		}
	}
	return query
}

func queryLimiter[T Viewer](r *http.Request, query *QueryStream[T], view *View) *QueryStream[T] {
	if view.Config.Query.Limit != nil {
		query = query.Limit(view.Config.Query.Limit)
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if view.Config.AllowLimit > 0 {
			if limitInt, err := strconv.Atoi(limit); err == nil && limitInt > 0 && limitInt <= view.Config.AllowLimit {
				return query.Limit(limitInt)
			}
		}
		query.Error = ErrorUnauthorized{}
	}
	return query
}

func queryOffsetter[T Viewer](r *http.Request, query *QueryStream[T], view *View) *QueryStream[T] {
	if view.Config.Query.Offset != nil {
		query = query.Offset(view.Config.Query.Offset)
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if view.Config.AllowOffset > 0 {
			if offsetInt, err := strconv.Atoi(offset); err == nil && offsetInt > 0 && offsetInt <= view.Config.AllowOffset {
				return query.Offset(offsetInt)
			}
		}
		query.Error = ErrorUnauthorized{}
	}
	return query
}

func querySorter[T Viewer](r *http.Request, query *QueryStream[T], view *View) *QueryStream[T] {
	if r.URL.Query().Has("sort") && len(view.Config.AllowSort) > 0 {
		_, sortWildcard := view.Config.AllowSort["*"]
		sortColumns := make([]string, 0)
		for _, sortColumn := range strings.Split(r.URL.Query().Get("sort"), ",") {
			s, _ := strings.CutPrefix(sortColumn, "-")
			if _, ok := view.Config.AllowSort[s]; ok || sortWildcard {
				sortColumns = append(sortColumns, sortColumn)
			} else {
				query.Error = ErrorUnauthorized{}
				return query
			}
		}
		if len(sortColumns) > 0 {
			return query.Sort(sortColumns...)
		}
	}
	return query
}

func recordGuard[T any](fieldsLower []string, fieldViews map[string]*View, data map[string]any) map[string]any {
	for key := range data {
		if !slices.Contains(fieldsLower, key) {
			delete(data, key)
		}
	}
	if len(fieldViews) > 0 {
		for fieldName, fieldView := range fieldViews {
			fieldLower := strings.ToLower(fieldName)
			if data[fieldLower] == nil {
				continue
			} else if rowData, ok := data[fieldLower].(map[string]any); ok {
				for rowKey := range rowData {
					rowFieldsLower := make([]string, 0)
					for rowField := range fieldView.Config.AllowFields {
						rowFieldsLower = append(rowFieldsLower, strings.ToLower(rowField))
					}
					if !slices.Contains(rowFieldsLower, rowKey) {
						delete(rowData, rowKey)
					}
				}
				data[fieldLower] = rowData
			} else if rowArrayData, ok := data[fieldLower].([]map[string]any); ok {
				for i, rowData := range rowArrayData {
					for rowKey := range rowData {
						rowFieldsLower := make([]string, 0)
						for rowField := range fieldView.Config.AllowFields {
							rowFieldsLower = append(rowFieldsLower, strings.ToLower(rowField))
						}
						if !slices.Contains(rowFieldsLower, rowKey) {
							delete(rowData, rowKey)
						}
					}
					rowArrayData[i] = rowData
				}
				data[fieldLower] = rowArrayData
			}
		}
	}
	return data
}

func requestToGuardContext(r *http.Request) context.Context {
	return context.WithValue(r.Context(), "Request", r)
}

func responseJson(w http.ResponseWriter, data any) {
	json, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(w, string(json))
}

func requestFieldViews(ctx context.Context, guards map[string]Viewer) map[string]*View {
	fieldViews := make(map[string]*View, len(guards))
	for fieldName, fieldGuard := range guards {
		fieldViews[fieldName] = fieldGuard.ViewSelect(ctx)
	}
	return fieldViews
}

// ShiftPath splits off the first component of p, which will be cleaned of
// relative components before processing. head will never contain a slash and
// tail will always be a rooted path without trailing slash.
// Via: https://blog.merovius.de/posts/2017-06-18-how-not-to-use-an-http-router/
func shiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}

func unwrap(handler func(*Strand) error) func(*Strand) error {
	return func(s *Strand) error {
		r := s.Request()
		_, r.URL.Path = shiftPath(r.URL.Path)
		s.Context = context.WithValue(s.Context, "Request", r)
		return handler(s)
	}
}

func UseDatabase(dbConnection *sql.DB) {
	db_ = dbConnection
}

func viewFields[T any](weave *Weave[T], view *View) ([]any, []string) {
	_, columsWildcard := view.Config.AllowFields["*"]
	var columns []any
	var fieldsLower []string

	for column, field := range weave.Fields {
		if _, ok := view.Config.AllowFields[field.Name]; ok || columsWildcard {
			if !strings.HasPrefix(field.Type.String(), "trance.OneToMany[") {
				columns = append(columns, column)
			}
			fieldsLower = append(fieldsLower, strings.ToLower(field.Name))
		}
	}
	return columns, fieldsLower
}
