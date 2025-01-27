package trance

import (
	"database/sql"
	"sort"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func TestModelScanMap(t *testing.T) {
	type testGroups struct {
		Id   int64  `@:"id" @primary:"true"`
		Name string `@:"name" @length:"100"`
	}
	type testAccounts struct {
		EditedAt sql.NullTime               `@:"edited_at"`
		Group    NullForeignKey[testGroups] `@:"group_id" @on_delete:"SET NULL"`
		Id       int64                      `@:"id" @primary:"true"`
		Name     string                     `@:"name"`
	}
	defer PurgeWeaves()
	model := UseWith[testAccounts](WeaveConfig{NoCache: true})

	data := []map[string]any{
		{
			"id":        int64(1),
			"name":      "foo",
			"edited_at": time.Date(2009, time.January, 2, 3, 0, 0, 0, time.UTC),
			"group_id":  int64(10),
		},
		{
			"id":        int64(2),
			"name":      "bar",
			"edited_at": nil,
			"group_id":  int64(20),
		},
		{
			"id":        int64(3),
			"name":      "baz",
			"edited_at": nil,
			"group_id":  nil,
		},
	}
	expected := []testAccounts{
		{
			Id:   1,
			Name: "foo",
			EditedAt: sql.NullTime{
				Time:  time.Date(2009, time.January, 2, 3, 0, 0, 0, time.UTC),
				Valid: true,
			},
			Group: NullForeignKey[testGroups]{
				Row:   &testGroups{Id: 10},
				Valid: true,
			},
		},
		{
			Id:   2,
			Name: "bar",
			Group: NullForeignKey[testGroups]{
				Row:   &testGroups{Id: 20},
				Valid: true,
			},
		},
		{Id: 3, Name: "baz"},
	}
	for i, row := range data {
		actual, err := model.ScanMap(row)
		if err != nil {
			t.Fatal("Unexpected error:", err)
		}
		if actual.Id != expected[i].Id ||
			actual.Name != expected[i].Name ||
			actual.EditedAt != expected[i].EditedAt ||
			actual.Group.Valid != expected[i].Group.Valid ||
			(actual.Group.Valid && *actual.Group.Row != *expected[i].Group.Row) {
			t.Errorf("Expected '%+v', got '%+v'", expected[i], actual)
		}
	}
}

type testGroupsModelToMap struct {
	Accounts OneToMany[testAccountsModelToMap] `@:"group_id"`
	Id       int64                             `@:"id" @primary:"true"`
	Name     string                            `@:"name" @length:"100"`
}
type testAccountsModelToMap struct {
	EditedAt sql.NullTime                         `@:"edited_at"`
	Group    NullForeignKey[testGroupsModelToMap] `@:"group_id" @on_delete:"SET NULL"`
	Id       int64                                `@:"id" @primary:"true"`
	Name     string                               `@:"name"`
}

func assertMapDeepEquals(t *testing.T, actual map[string]any, expected map[string]any) {
	actualKeys := maps.Keys(actual)
	expectKeys := maps.Keys(expected)
	sort.Strings(actualKeys)
	sort.Strings(expectKeys)
	if !slices.Equal(actualKeys, expectKeys) {
		t.Errorf("Expected '%#v', got '%#v'", expected, actual)
	}
	for _, key := range actualKeys {
		actualValue := actual[key]
		expectValue := expected[key]

		switch av := actualValue.(type) {
		case []map[string]any:
			ev, ok := expectValue.([]map[string]any)
			if !ok {
				t.Errorf("Expected\n'%#v', got\n'%#v'", expected, actual)
			}
			for i := range av {
				assertMapDeepEquals(t, av[i], ev[i])
			}
		case map[string]any:
			ev, ok := expectValue.(map[string]any)
			if !ok {
				t.Errorf("Expected\n'%#v', got\n'%#v'", expected, actual)
			}
			assertMapDeepEquals(t, av, ev)
		default:
			if av != expectValue {
				t.Errorf("Expected '%#v', got '%#v'", expected, actual)
			}
		}
	}
}

func TestModelToJsonMap(t *testing.T) {
	defer func() {
		PurgeWeaves()
	}()
	PurgeWeaves()
	model := UseWith[testAccountsModelToMap](WeaveConfig{NoCache: true})

	rows := []testAccountsModelToMap{
		{
			Id:   1,
			Name: "foo",
			EditedAt: sql.NullTime{
				Time:  time.Date(2009, time.January, 2, 3, 0, 0, 0, time.UTC),
				Valid: true,
			},
			Group: NullForeignKey[testGroupsModelToMap]{
				Row:   &testGroupsModelToMap{Id: 10},
				Valid: true,
			},
		},
		{
			Id:   2,
			Name: "bar",
			Group: NullForeignKey[testGroupsModelToMap]{
				Row:   &testGroupsModelToMap{Id: 20},
				Valid: true,
			},
		},
		{Id: 3, Name: "baz"},
	}
	expected := []map[string]any{
		{
			"id":       int64(1),
			"name":     "foo",
			"editedat": time.Date(2009, time.January, 2, 3, 0, 0, 0, time.UTC),
			"group": map[string]any{
				"accounts": []map[string]any{},
				"name":     "",
				"id":       int64(10),
			},
		},
		{
			"id":       int64(2),
			"name":     "bar",
			"editedat": nil,
			"group": map[string]any{
				"accounts": []map[string]any{},
				"name":     "",
				"id":       int64(20),
			},
		},
		{
			"id":       int64(3),
			"name":     "baz",
			"editedat": nil,
			"group":    nil,
		},
	}

	for i, row := range rows {
		actual := model.ToJsonMap(&row)
		assertMapDeepEquals(t, actual, expected[i])
	}

	groupsModel := UseWith[testGroupsModelToMap](WeaveConfig{NoCache: true})
	groups := []testGroupsModelToMap{
		{
			Id:   1,
			Name: "foo",
		},
		{
			Id:   2,
			Name: "bar",
		},
	}
	expected = []map[string]any{
		{
			"accounts": []map[string]any{},
			"id":       int64(1),
			"name":     "foo",
		},
		{
			"accounts": []map[string]any{},
			"id":       int64(2),
			"name":     "bar",
		},
	}
	for i, row := range groups {
		actual := groupsModel.ToJsonMap(&row)
		assertMapDeepEquals(t, actual, expected[i])
	}
}

func TestModelToMap(t *testing.T) {
	defer func() {
		PurgeWeaves()
	}()
	PurgeWeaves()
	model := UseWith[testAccountsModelToMap](WeaveConfig{NoCache: true})

	rows := []testAccountsModelToMap{
		{
			Id:   1,
			Name: "foo",
			EditedAt: sql.NullTime{
				Time:  time.Date(2009, time.January, 2, 3, 0, 0, 0, time.UTC),
				Valid: true,
			},
			Group: NullForeignKey[testGroupsModelToMap]{
				Row:   &testGroupsModelToMap{Id: 10},
				Valid: true,
			},
		},
		{
			Id:   2,
			Name: "bar",
			Group: NullForeignKey[testGroupsModelToMap]{
				Row:   &testGroupsModelToMap{Id: 20},
				Valid: true,
			},
		},
		{Id: 3, Name: "baz"},
	}
	expected := []map[string]any{
		{
			"id":        int64(1),
			"name":      "foo",
			"edited_at": time.Date(2009, time.January, 2, 3, 0, 0, 0, time.UTC),
			"group_id":  int64(10),
		},
		{
			"id":        int64(2),
			"name":      "bar",
			"edited_at": nil,
			"group_id":  int64(20),
		},
		{
			"id":        int64(3),
			"name":      "baz",
			"edited_at": nil,
			"group_id":  nil,
		},
	}
	for i, row := range rows {
		actual, err := model.ToMap(&row)
		if err != nil {
			t.Fatal("Unexpected error:", err)
		}
		if !maps.Equal(actual, expected[i]) {
			t.Errorf("Expected '%#v', got '%#v'", expected[i], actual)
		}
	}

	groupsModel := UseWith[testGroupsModelToMap](WeaveConfig{NoCache: true})
	groups := []testGroupsModelToMap{
		{
			Id:   1,
			Name: "foo",
		},
		{
			Id:   2,
			Name: "bar",
		},
	}
	expected = []map[string]any{
		{
			"id":   int64(1),
			"name": "foo",
		},
		{
			"id":   int64(2),
			"name": "bar",
		},
	}
	for i, row := range groups {
		actual, err := groupsModel.ToMap(&row)
		if err != nil {
			t.Fatal("Unexpected error:", err)
		}
		if !maps.Equal(actual, expected[i]) {
			t.Errorf("Expected '%#v', got '%#v'", expected[i], actual)
		}
	}
}

func TestRegister(t *testing.T) {
	type testModel struct {
		Id   int64  `@:"id" @primary:"true"`
		Name string `@:"name"`
	}
	defer PurgeWeaves()
	PurgeWeaves()
	m1 := Use[testModel]()
	m2 := Use[testModel]()
	m3 := UseWith[testModel](WeaveConfig{Table: "testmodelwithconfig"})
	m4 := UseWith[testModel](WeaveConfig{NoCache: true})
	m5 := UseWith[testModel](WeaveConfig{NoCache: true})
	if m1 == m3 || m2 == m3 {
		t.Errorf("Expected '%#v' to be different from '%#v' and '%#v'", m3, m1, m2)
	}
	if m1 != m2 {
		t.Errorf("Expected '%#v', got '%#v'", m1, m2)
	}
	if m1 == m4 {
		t.Errorf("Expected '%#v' to be different from '%#v'", m1, m4)
	}
	if m4 == m5 {
		t.Errorf("Expected '%#v' to be different from '%#v'", m4, m5)
	}
	if m1.Table != "testmodel" {
		t.Errorf("Expected '%s' to be 'testmodel'", m1.Table)
	}
	if m3.Table != "testmodelwithconfig" {
		t.Errorf("Expected '%s' to be 'testmodelwithconfig'", m3.Table)
	}
	if !m4.Config.NoCache {
		t.Errorf("Expected '%#v' to be '%#v'", m4.Config.NoCache, true)
	}
}

func TestScanToMap(t *testing.T) {
	type testAccounts struct {
		EditedAt sql.NullTime `@:"edited_at"`
		Id       int64        `@:"id" @primary:"true"`
		Name     string       `@:"name"`
	}
	defer PurgeWeaves()
	accounts := QueryWith[testAccounts](WeaveConfig{NoCache: true})

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal("failed to open sqlmock database:", err)
	}
	defer db.Close()

	UseDatabase(db)

	rows := sqlmock.NewRows([]string{"id", "name", "edited_at"}).
		AddRow(1, "foo", time.Date(2009, time.January, 2, 3, 0, 0, 0, time.UTC)).
		AddRow(2, "bar", nil).
		AddRow(3, "baz", nil)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	rs, _ := db.Query("SELECT")
	defer rs.Close()

	expected := []map[string]any{
		{
			"id":        int64(1),
			"name":      "foo",
			"edited_at": time.Date(2009, time.January, 2, 3, 0, 0, 0, time.UTC),
		},
		{
			"id":        int64(2),
			"name":      "bar",
			"edited_at": nil,
		},
		{
			"id":        int64(3),
			"name":      "baz",
			"edited_at": nil,
		},
	}
	i := 0
	for rs.Next() {
		row, err := accounts.ScanToMap(rs)
		if err != nil {
			t.Fatal("Unexpected error:", err)
		}
		if !maps.Equal(row, expected[i]) {
			t.Errorf("Expected '%#v', got '%#v'", expected[i], row)
		}
		i++
	}
}

func TestUse(t *testing.T) {
	type testAccounts struct {
		EditedAt sql.NullTime `@:"edited_at"`
		Id       int64        `@:"id" @primary:"true"`
		Name     string       `@:"name"`
	}
	type testGroups struct {
		Id       int64                   `@:"id" @primary:"true"`
		Name     string                  `@:"name" @length:"100"`
		Accounts OneToMany[testAccounts] `@:"group_id"`
	}
	defer PurgeWeaves()
	groups := UseWith[testGroups](WeaveConfig{NoCache: true})
	columns := maps.Keys(groups.Fields)
	sort.Strings(columns)
	expectedColumns := []string{"Accounts", "id", "name"}
	expectedPrimaryColumn := "id"
	expectedPrimaryField := "Id"
	expectedTable := "testgroups"
	if !slices.Equal(columns, expectedColumns) {
		t.Errorf("Expected '%+v', got '%+v'", expectedColumns, columns)
	}
	if groups.PrimaryColumn != expectedPrimaryColumn {
		t.Errorf("Expected '%s', got '%s'", expectedPrimaryColumn, groups.PrimaryColumn)
	}
	if groups.PrimaryField != expectedPrimaryField {
		t.Errorf("Expected '%s', got '%s'", expectedPrimaryField, groups.PrimaryField)
	}
	if groups.Table != expectedTable {
		t.Errorf("Expected '%s', got '%s'", expectedTable, groups.Table)
	}
}
