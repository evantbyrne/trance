package mysqldialect

import (
	"database/sql"
	"sort"
	"testing"
	"time"

	"github.com/evantbyrne/trance"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func TestAs(t *testing.T) {
	dialect := MysqlDialect{}
	expected := map[string]trance.SqlAs{
		"`x` AS `alias1`":        trance.As("x", "alias1"),
		"`x` AS `y` AS `alias2`": trance.As(trance.As("x", "y"), "alias2"),
		"count(*) AS `alias3`":   trance.As(trance.Unsafe("count(*)"), "alias3"),
	}
	for expected, alias := range expected {
		sql := alias.StringForDialect(dialect)
		if expected != sql {
			t.Errorf("Expected '%+v', got '%+v'", expected, sql)
		}
	}
}

func TestColumn(t *testing.T) {
	dialect := MysqlDialect{}
	expected := map[string]trance.SqlColumn{
		"`x`":         trance.Column("x"),
		"`x`.`y`":     trance.Column("x.y"),
		"`x`.`y`.`z`": trance.Column("x.y.z"),
		"`x```":       trance.Column("x`"),
	}
	for expected, column := range expected {
		sql := column.StringForDialect(dialect)
		if expected != sql {
			t.Errorf("Expected '%+v', got '%+v'", expected, sql)
		}
	}
}

func TestBuildDelete(t *testing.T) {
	type testModel struct {
		Id     int64  `@:"test_id" @primary:"true"`
		Value1 string `@:"test_value_1" @length:"100"`
		Value2 string `@:"test_value_2" @length:"100"`
	}
	defer trance.PurgeWeaves()

	dialect := MysqlDialect{}
	weave := trance.UseWith[testModel](trance.WeaveConfig{NoCache: true})

	query := trance.QueryWith[testModel](trance.WeaveConfig{NoCache: true})
	config := query.Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs := []any{}
	expectedSql := "DELETE FROM `testmodel`"
	queryString, args, err := dialect.BuildDelete(config)
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}

	// WHERE
	query.Filter("test_id", "=", 1)
	config = query.Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs = []any{1}
	expectedSql = "DELETE FROM `testmodel` WHERE `test_id` = ?"
	queryString, args, err = dialect.BuildDelete(config)
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}

	// ORDER BY
	query.Sort("-test_id")
	config = query.Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedSql = "DELETE FROM `testmodel` WHERE `test_id` = ? ORDER BY `test_id` DESC"
	queryString, args, err = dialect.BuildDelete(config)
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}

	// LIMIT
	query.Limit(3)
	config = query.Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs = []any{1, 3}
	expectedSql = "DELETE FROM `testmodel` WHERE `test_id` = ? ORDER BY `test_id` DESC LIMIT ?"
	queryString, args, err = dialect.BuildDelete(config)
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}
}

func TestBuildInsert(t *testing.T) {
	type testModel struct {
		Id     int64  `@:"test_id" @primary:"true"`
		Value1 string `@:"test_value_1" @length:"100"`
		Value2 string `@:"test_value_2" @length:"100"`
	}
	defer trance.PurgeWeaves()

	dialect := MysqlDialect{}
	weave := trance.UseWith[testModel](trance.WeaveConfig{NoCache: true})

	config := trance.QueryWith[testModel](trance.WeaveConfig{NoCache: true}).Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs := []any{"foo", "bar"}
	expectedSql := "INSERT INTO `testmodel` (`test_value_1`,`test_value_2`) VALUES (?,?)"
	queryString, args, err := dialect.BuildInsert(config, map[string]any{
		"test_value_1": "foo",
		"test_value_2": "bar",
	}, "test_value_1", "test_value_2")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}
}

func TestBuildSelect(t *testing.T) {
	type testModel struct {
		Id     int64  `@:"test_id" @primary:"true"`
		Value1 string `@:"test_value_1" @length:"100"`
		Value2 string `@:"test_value_2" @length:"100"`
	}
	defer trance.PurgeWeaves()

	dialect := MysqlDialect{}
	weave := trance.UseWith[testModel](trance.WeaveConfig{NoCache: true})

	config := trance.QueryWith[testModel](trance.WeaveConfig{NoCache: true}).Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs := []any{}
	expectedSql := "SELECT * FROM `testmodel`"
	queryString, args, err := dialect.BuildSelect(config)
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}

	// SELECT
	config = trance.QueryWith[testModel](trance.WeaveConfig{NoCache: true}).Select("id", "value1", trance.Unsafe("count(1) as `count`"), trance.As("value2", "value3")).Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs = []any{}
	expectedSql = "SELECT `id`,`value1`,count(1) as `count`,`value2` AS `value3` FROM `testmodel`"
	queryString, args, err = dialect.BuildSelect(config)
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}

	// WHERE
	config = trance.QueryWith[testModel](trance.WeaveConfig{NoCache: true}).Filter("id", "=", 1).Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs = []any{1}
	expectedSql = "SELECT * FROM `testmodel` WHERE `id` = ?"
	queryString, args, err = dialect.BuildSelect(config)
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}

	config = trance.QueryWith[testModel](trance.WeaveConfig{NoCache: true}).Filter("id", "IN", trance.Sql(trance.Param(1), ",", trance.Param(2))).Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs = []any{1, 2}
	expectedSql = "SELECT * FROM `testmodel` WHERE `id` IN (?,?)"
	queryString, args, err = dialect.BuildSelect(config)
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}

	// JOIN
	config = trance.QueryWith[testModel](trance.WeaveConfig{NoCache: true}).Select(trance.Unsafe("*")).Join("groups", trance.Or(
		trance.Q("groups.id", "=", trance.Column("accounts.group_id")),
		trance.Q("groups.id", "IS", nil))).Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs = []any{}
	expectedSql = "SELECT * FROM `testmodel` INNER JOIN `groups` ON ( `groups`.`id` = `accounts`.`group_id` OR `groups`.`id` IS NULL )"
	queryString, args, err = dialect.BuildSelect(config)
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}

	// SORT
	config = trance.QueryWith[testModel](trance.WeaveConfig{NoCache: true}).Sort("test_id", "-test_value_1").Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs = []any{}
	expectedSql = "SELECT * FROM `testmodel` ORDER BY `test_id` ASC, `test_value_1` DESC"
	queryString, args, err = dialect.BuildSelect(config)
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}

	// LIMIT and OFFSET
	config = trance.QueryWith[testModel](trance.WeaveConfig{NoCache: true}).Filter("id", "=", 1).Offset(20).Limit(10).Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs = []any{1, 10, 20}
	expectedSql = "SELECT * FROM `testmodel` WHERE `id` = ? LIMIT ? OFFSET ?"
	queryString, args, err = dialect.BuildSelect(config)
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}
}

func TestBuildTableColumnAdd(t *testing.T) {
	type testModel struct {
		Value string `@:"test_value" @length:"100"`
	}
	defer func() {
		trance.PurgeWeaves()
	}()
	trance.PurgeWeaves()

	dialect := MysqlDialect{}
	weave := trance.Use[testModel]()
	config := trance.QueryConfig{
		Fields: weave.Fields,
		Table:  "testmodel",
	}
	expectedSql := "ALTER TABLE `testmodel` ADD COLUMN `test_value` VARCHAR(100) NOT NULL"
	queryString, err := dialect.BuildTableColumnAdd(config, "test_value")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
}

func TestBuildTableColumnDrop(t *testing.T) {
	dialect := MysqlDialect{}
	config := trance.QueryConfig{Table: "testmodel"}
	expectedSql := "ALTER TABLE `testmodel` DROP COLUMN `test_value`"
	queryString, err := dialect.BuildTableColumnDrop(config, "test_value")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
}

func TestBuildTableCreate(t *testing.T) {
	type testModel struct {
		Id     int64  `@:"test_id" @primary:"true"`
		Value1 string `@:"test_value_1" @length:"100"`
	}
	defer func() {
		trance.PurgeWeaves()
	}()
	trance.PurgeWeaves()

	dialect := MysqlDialect{}
	weave := trance.Use[testModel]()
	config := trance.QueryConfig{
		Fields: weave.Fields,
		Table:  "testmodel",
	}
	expectedSql := "CREATE TABLE `testmodel` (\n" +
		"\t`test_id` BIGINT PRIMARY KEY NOT NULL AUTO_INCREMENT,\n" +
		"\t`test_value_1` VARCHAR(100) NOT NULL\n" +
		")"
	queryString, err := dialect.BuildTableCreate(config, trance.TableCreateConfig{})
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}

	expectedSql = "CREATE TABLE IF NOT EXISTS `testmodel` (\n" +
		"\t`test_id` BIGINT PRIMARY KEY NOT NULL AUTO_INCREMENT,\n" +
		"\t`test_value_1` VARCHAR(100) NOT NULL\n" +
		")"
	queryString, err = dialect.BuildTableCreate(config, trance.TableCreateConfig{IfNotExists: true})
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
}

func TestBuildTableDrop(t *testing.T) {
	dialect := MysqlDialect{}
	config := trance.QueryConfig{Table: "testmodel"}
	expectedSql := "DROP TABLE `testmodel`"
	queryString, err := dialect.BuildTableDrop(config, trance.TableDropConfig{})
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}

	expectedSql = "DROP TABLE IF EXISTS `testmodel`"
	queryString, err = dialect.BuildTableDrop(config, trance.TableDropConfig{IfExists: true})
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
}

func TestBuildUpdate(t *testing.T) {
	type testModel struct {
		Id     int64  `@:"test_id" @primary:"true"`
		Value1 string `@:"test_value_1" @length:"100"`
		Value2 string `@:"test_value_2" @length:"100"`
	}
	defer trance.PurgeWeaves()

	dialect := MysqlDialect{}
	weave := trance.Use[testModel]()

	query := trance.QueryWith[testModel](trance.WeaveConfig{NoCache: true}).Filter("test_id", "=", 1)
	config := query.Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs := []any{"foo", "bar", 1}
	expectedSql := "UPDATE `testmodel` SET `test_value_1` = ?,`test_value_2` = ? WHERE `test_id` = ?"
	queryString, args, err := dialect.BuildUpdate(config, map[string]any{
		"id":           123,
		"test_value_1": "foo",
		"test_value_2": "bar",
	}, "test_value_1", "test_value_2")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}

	query.Limit(3)
	config = query.Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedArgs = []any{"foo", "bar", 1, 3}
	expectedSql = "UPDATE `testmodel` SET `test_value_1` = ?,`test_value_2` = ? WHERE `test_id` = ? LIMIT ?"
	queryString, args, err = dialect.BuildUpdate(config, map[string]any{
		"id":           123,
		"test_value_1": "foo",
		"test_value_2": "bar",
	}, "test_value_1", "test_value_2")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}

	query.Sort("-test_id")
	config = query.Config
	config.Fields = weave.Fields
	config.Table = "testmodel"
	expectedSql = "UPDATE `testmodel` SET `test_value_1` = ?,`test_value_2` = ? WHERE `test_id` = ? ORDER BY `test_id` DESC LIMIT ?"
	queryString, args, err = dialect.BuildUpdate(config, map[string]any{
		"id":           123,
		"test_value_1": "foo",
		"test_value_2": "bar",
	}, "test_value_1", "test_value_2")
	if err != nil {
		t.Errorf("Unexpected error %s", err.Error())
	}
	if queryString != expectedSql {
		t.Errorf("Expected '%s', got '%s'", expectedSql, queryString)
	}
	if !slices.Equal(args, expectedArgs) {
		t.Errorf("Expected '%s', got '%s'", expectedArgs, args)
	}
}

func TestColumnType(t *testing.T) {
	type testFkInt struct {
		Id int64 `@:"id" @primary:"true"`
	}

	type testFkString struct {
		Id string `@:"id" @primary:"true" @length:"100"`
	}

	type testModel struct {
		BigInt         int64                            `@:"test_big_int"`
		BigIntNull     sql.NullInt64                    `@:"test_big_int_null"`
		Bool           bool                             `@:"test_bool"`
		BoolNull       sql.NullBool                     `@:"test_bool_null"`
		Custom         []byte                           `@:"test_custom" @type:"BINARY(128) NOT NULL"`
		Default        string                           `@:"test_default" @default:"'foo'" @length:"100"`
		Float          float32                          `@:"test_float"`
		Double         float64                          `@:"test_double"`
		DoubleNull     sql.NullFloat64                  `@:"test_double_null"`
		Id             int64                            `@:"test_id" @primary:"true"`
		Int            int32                            `@:"test_int"`
		IntNull        sql.NullInt32                    `@:"test_int_null"`
		SmallInt       int16                            `@:"test_small_int"`
		SmallIntNull   sql.NullInt16                    `@:"test_small_int_null"`
		Text           string                           `@:"test_text"`
		TextNull       sql.NullString                   `@:"test_text_null"`
		Time           time.Time                        `@:"test_time"`
		TimeNow        time.Time                        `@:"test_time_now" @default:"now()"`
		TimeNull       sql.NullTime                     `@:"test_time_null"`
		TinyInt        int8                             `@:"test_tiny_int"`
		Varchar        string                           `@:"test_varchar" @length:"100"`
		VarcharNull    sql.NullString                   `@:"test_varchar_null" @length:"50"`
		ForiegnKey     trance.ForeignKey[testFkString]  `@:"test_fk_id" @on_delete:"CASCADE"`
		ForiegnKeyNull trance.NullForeignKey[testFkInt] `@:"test_fk_null_id" @on_delete:"SET NULL" @on_update:"SET NULL"`
		Unique         string                           `@:"test_unique" @length:"255" @unique:"true"`
	}
	defer func() {
		trance.PurgeWeaves()
	}()
	trance.PurgeWeaves()

	expected := map[string]string{
		"test_big_int":        "BIGINT NOT NULL",
		"test_big_int_null":   "BIGINT NULL",
		"test_bool":           "BOOLEAN NOT NULL",
		"test_bool_null":      "BOOLEAN NULL",
		"test_custom":         "BINARY(128) NOT NULL",
		"test_default":        "VARCHAR(100) NOT NULL DEFAULT 'foo'",
		"test_float":          "FLOAT NOT NULL",
		"test_double":         "DOUBLE NOT NULL",
		"test_double_null":    "DOUBLE NULL",
		"test_id":             "BIGINT PRIMARY KEY NOT NULL AUTO_INCREMENT",
		"test_int":            "INTEGER NOT NULL",
		"test_int_null":       "INTEGER NULL",
		"test_small_int":      "SMALLINT NOT NULL",
		"test_small_int_null": "SMALLINT NULL",
		"test_time":           "DATETIME NOT NULL",
		"test_time_now":       "DATETIME NOT NULL DEFAULT now()",
		"test_time_null":      "DATETIME NULL",
		"test_text":           "TEXT NOT NULL",
		"test_text_null":      "TEXT NULL",
		"test_tiny_int":       "TINYINT NOT NULL",
		"test_varchar":        "VARCHAR(100) NOT NULL",
		"test_varchar_null":   "VARCHAR(50) NULL",
		"test_fk_id":          "VARCHAR(100) NOT NULL REFERENCES `testfkstring` (`id`) ON DELETE CASCADE",
		"test_fk_null_id":     "BIGINT NULL REFERENCES `testfkint` (`id`) ON UPDATE SET NULL ON DELETE SET NULL",
		"test_unique":         "VARCHAR(255) NOT NULL UNIQUE",
	}

	dialect := MysqlDialect{}
	weave := trance.Use[testModel]()
	fieldKeys := maps.Keys(weave.Fields)
	sort.Strings(fieldKeys)

	for _, fieldName := range fieldKeys {
		field := weave.Fields[fieldName]
		columnType, err := dialect.ColumnType(field)
		if err != nil {
			t.Fatalf(`dialect.ColumnType() threw error for '%#v': %s`, field, err)
		}
		if columnType != expected[fieldName] {
			t.Fatalf(`dialect.ColumnType() returned '%s', but expected '%s' for '%#v'`, columnType, expected[fieldName], field)
		}
	}
}

func TestQuoteIdentifier(t *testing.T) {
	values := map[string]string{
		"abc":    "`abc`",
		"a`bc":   "`a``bc`",
		"a``b`c": "`a````b``c`",
		"`abc":   "```abc`",
		"abc`":   "`abc```",
		"ab\\`c": "`ab\\``c`",
		"abc\\":  "`abc\\`",
	}

	for identifier, expected := range values {
		actual := QuoteIdentifier(identifier)
		if actual != expected {
			t.Errorf("Expected %s, got %s", expected, actual)
		}
	}
}
