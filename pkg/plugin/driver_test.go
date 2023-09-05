package plugin_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cadl/grafana-databend-datasource/pkg/converters"
	"github.com/cadl/grafana-databend-datasource/pkg/plugin"
	godatabend "github.com/databendcloud/databend-go"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
)

const databendPort = 8000
const databendHost = "localhost"
const databendUsername = "databend"
const databendPassword = "databend"

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
func TestMain(m *testing.M) {
	useDocker := strings.ToLower(getEnv("DATABEND_USE_DOCKER", "true"))
	if useDocker == "false" {
		fmt.Printf("Using external Databend for IT tests -  %s:%d\n",
			databendHost, databendPort)
		os.Exit(m.Run())
	}
	// create a databend container
	ctx := context.Background()
	// SET OS ENV VAR DATABEND_PORT to override default port
	os.Setenv("DATABEND_PORT", fmt.Sprintf("%d", databendPort))

	compose, err := tc.NewDockerCompose("../../docker-compose.yml")
	if err != nil {
		panic(err)
	}
	defer compose.Down(ctx, tc.RemoveOrphans(true), tc.RemoveImagesLocal)
	err = compose.Up(ctx, tc.Wait(true))
	if err != nil {
		panic(err)
	}
	m.Run()
}
func TestConnect(t *testing.T) {
	port := databendPort
	host := databendHost
	username := databendUsername
	password := databendPassword
	queryTimeoutNumber := 3600
	queryTimeoutString := "3600"
	databend := plugin.Databend{}
	t.Run("should not error when valid settings passed", func(t *testing.T) {
		secure := map[string]string{}
		secure["password"] = password
		settings := backend.DataSourceInstanceSettings{JSONData: []byte(fmt.Sprintf(`{ "server": "%s", "port": %d, "username": "%s", "queryTimeout": "%s"}`, host, port, username, queryTimeoutString)), DecryptedSecureJSONData: secure}
		_, err := databend.Connect(settings, json.RawMessage{})
		assert.Equal(t, nil, err)
	})
	t.Run("should not error when valid settings passed - with query timeout as number", func(t *testing.T) {
		secure := map[string]string{}
		secure["password"] = password
		settings := backend.DataSourceInstanceSettings{JSONData: []byte(fmt.Sprintf(`{ "server": "%s", "port": %d, "username": "%s", "queryTimeout": %d }`, host, port, username, queryTimeoutNumber)), DecryptedSecureJSONData: secure}
		_, err := databend.Connect(settings, json.RawMessage{})
		assert.Equal(t, nil, err)
	})
}
func TestHTTPConnect(t *testing.T) {
	port := databendPort
	host := databendHost
	username := databendUsername
	password := databendPassword
	databend := plugin.Databend{}
	t.Run("should not error when valid settings passed", func(t *testing.T) {
		secure := map[string]string{}
		secure["password"] = password
		settings := backend.DataSourceInstanceSettings{JSONData: []byte(fmt.Sprintf(`{ "server": "%s", "port": %d, "username": "%s"}`, host, port, username)), DecryptedSecureJSONData: secure}
		_, err := databend.Connect(settings, json.RawMessage{})
		assert.Equal(t, nil, err)
	})
}
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
func setupConnection(t *testing.T) *sql.DB {
	cfg := godatabend.Config{
		Host:     fmt.Sprintf("%s:%d", databendHost, databendPort),
		User:     databendUsername,
		Password: databendPassword,
		Database: "default",
		SSLMode:  godatabend.SSL_MODE_DISABLE,
		Location: time.UTC,
	}

	db, err := sql.Open("databend", cfg.FormatDSN())
	if err != nil {
		panic(err)
	}
	return db
}
func setupTest(t *testing.T, ddl string) (*sql.DB, func(t *testing.T)) {
	conn := setupConnection(t)
	_, err := conn.Exec("DROP TABLE IF EXISTS simple_table")
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE table simple_table(%s);", ddl))
	require.NoError(t, err)
	return conn, func(t *testing.T) {
		_, err := conn.Exec("DROP TABLE simple_table")
		require.NoError(t, err)
	}
}
func insertData(t *testing.T, conn *sql.DB, data ...interface{}) {
	scope, err := conn.Begin()
	require.NoError(t, err)

	batch, err := scope.Prepare("INSERT INTO simple_table values")
	require.NoError(t, err)
	for _, val := range data {
		if val == nil {
			_, err = batch.Exec(nil)
		} else {
			switch reflect.ValueOf(val).Kind() {
			case reflect.Map, reflect.Slice:
				jsonBytes, _ := json.Marshal(val)
				_, err = batch.Exec(string(jsonBytes))
			default:
				_, err = batch.Exec(val)
			}
		}
		require.NoError(t, err)
	}
	require.NoError(t, scope.Commit())
}
func toJson(obj interface{}) (json.RawMessage, error) {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.New("unable to marshal")
	}
	var rawJSON json.RawMessage
	err = json.Unmarshal(bytes, &rawJSON)
	if err != nil {
		return nil, errors.New("unable to unmarshal")
	}
	return rawJSON, nil
}
func checkFieldValue(t *testing.T, field *data.Field, expected ...interface{}) {
	for i, eVal := range expected {
		val := field.At(i)
		if eVal == nil {
			assert.Nil(t, val)
			return
		}
		switch tVal := eVal.(type) {
		case float64:
			assert.InDelta(t, tVal, val, 0.01)
		default:
			switch reflect.ValueOf(eVal).Kind() {
			case reflect.Map, reflect.Slice:
				jsonRaw, err := toJson(tVal)
				assert.Nil(t, err)
				assert.Equal(t, jsonRaw, *val.(*json.RawMessage))
				return
			}
			assert.Equal(t, eVal, val)
		}
	}
}

func checkRows(t *testing.T, conn *sql.DB, rowLimit int64, expectedValues ...interface{}) {
	rows, err := conn.Query(fmt.Sprintf("SELECT * FROM simple_table LIMIT %d", rowLimit))
	require.NoError(t, err)
	types, _ := rows.ColumnTypes()
	for _, col := range types {
		fmt.Println(col.Name(), col.DatabaseTypeName())
	}
	frame, err := sqlutil.FrameFromRows(rows, rowLimit, converters.DatabendConverters...)
	require.NoError(t, err)
	assert.Equal(t, 1, len(frame.Fields))
	checkFieldValue(t, frame.Fields[0], expectedValues...)
}

//	func TestConvertNullableInt8(t *testing.T) {
//		t.Run("TestConvertNullableInt8", func(t *testing.T) {
//			conn, close := setupTest(t, "col1 INT8 NULL")
//			defer close(t)
//			val := int8(9)
//			insertData(t, conn, val, nil)
//			checkRows(t, conn, 2, &val, nil)
//		})
//	}
//
//	func TestConvertNullableInt16(t *testing.T) {
//		t.Run("TestConvertNullableInt16", func(t *testing.T) {
//			conn, close := setupTest(t, "col1 INT16 NULL")
//			defer close(t)
//			val := int16(10)
//			insertData(t, conn, val, nil)
//			checkRows(t, conn, 2, &val, nil)
//		})
//	}
//
//	func TestConvertNullableInt32(t *testing.T) {
//		t.Run("TestConvertNullableInt32", func(t *testing.T) {
//			conn, close := setupTest(t, "col1 INT32 NULL")
//			defer close(t)
//			val := int32(11)
//			insertData(t, conn, val, nil)
//			checkRows(t, conn, 2, &val, nil)
//		})
//	}
//
//	func TestConvertNullableInt64(t *testing.T) {
//		t.Run(t.Name(), func(t *testing.T) {
//			conn, close := setupTest(t, "col1 INT64 NULL")
//			defer close(t)
//			val := int64(12)
//			insertData(t, conn, val, nil)
//			checkRows(t, conn, 2, &val, nil)
//		})
//	}
func TestConvertInt8(t *testing.T) {
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 INT8")
		defer close(t)
		val := int8(126)
		insertData(t, conn, val)
		checkRows(t, conn, 1, val)
	})
}
func TestConvertInt16(t *testing.T) {
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 INT16")
		defer close(t)
		val := int16(32767)
		insertData(t, conn, val)
		checkRows(t, conn, 1, val)
	})
}
func TestConvertInt32(t *testing.T) {
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 INT32")
		defer close(t)
		val := int32(2147483647)
		insertData(t, conn, val)
		checkRows(t, conn, 1, val)
	})
}

// todo: fix this test
//
//	func TestConvertInt64(t *testing.T) {
//		t.Run(t.Name(), func(t *testing.T) {
//			conn, close := setupTest(t, "col1 INT64")
//			defer close(t)
//			insertData(t, conn, int64(9223372036854775806))
//			checkRows(t, conn, 1, int64(9223372036854775806))
//		})
//	}
func TestConvertFloat32(t *testing.T) {
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 FLOAT32")
		defer close(t)
		insertData(t, conn, float32(17.1))
		checkRows(t, conn, 1, float32(17.1))
	})
}
func TestConvertFloat64(t *testing.T) {
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 FLOAT64")
		defer close(t)
		insertData(t, conn, float64(18.1))
		checkRows(t, conn, 1, float64(18.1))
	})
}

// func TestConvertNullableFloat32(t *testing.T) {

// 	t.Run(t.Name(), func(t *testing.T) {
// 		conn, close := setupTest(t, "col1 FLOAT32 NULL")
// 		defer close(t)
// 		val := float32(19.1)
// 		insertData(t, conn, val, nil)
// 		checkRows(t, conn, 2, &val, nil)
// 	})

// }

// func TestConvertNullableFloat64(t *testing.T) {

//		t.Run(t.Name(), func(t *testing.T) {
//			conn, close := setupTest(t, "col1 FLOAT64 NULL")
//			defer close(t)
//			val := float64(20.1)
//			insertData(t, conn, val, nil)
//			checkRows(t, conn, 2, &val, nil)
//		})
//	}
func TestConvertBool(t *testing.T) {
	var expected interface{} = true
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 BOOL")
		defer close(t)
		insertData(t, conn, true)
		checkRows(t, conn, 1, expected)
	})
}

// func TestConvertNullableBool(t *testing.T) {
// 	t.Run(t.Name(), func(t *testing.T) {
// 		conn, close := setupTest(t, "col1 BOOL NULL")
// 		defer close(t)
// 		val := true
// 		insertData(t, conn, val, nil)
// 		checkRows(t, conn, 2, &val, nil)
// 	})
// }

var date, _ = time.ParseInLocation("2006-01-02", "2022-01-12", time.UTC)

func TestConvertDate(t *testing.T) {
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 Date")
		defer close(t)
		insertData(t, conn, date)
		checkRows(t, conn, 1, date)
	})
}

// func TestConvertNullableDate(t *testing.T) {
// 	t.Run(t.Name(), func(t *testing.T) {
// 		conn, close := setupTest(t, "col1 Date NULL")
// 		defer close(t)
// 		insertData(t, conn, date, nil)
// 		checkRows(t, conn, 2, &date, nil)
// 	})
// }

var datetime, _ = time.Parse("2006-01-02 15:04:05", "2022-01-12 00:00:00")

func TestConvertDateTime(t *testing.T) {
	localtime := datetime.In(time.UTC)
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 DateTime")
		defer close(t)
		insertData(t, conn, localtime.Format(time.RFC3339))
		checkRows(t, conn, 1, localtime)
	})
}

// func TestConvertNullableDateTime(t *testing.T) {
// 	t.Run(t.Name(), func(t *testing.T) {
// 		conn, close := setupTest(t, "col1 DateTime NULL")
// 		defer close(t)
// 		insertData(t, conn, datetime, nil)
// 		checkRows(t, conn, 2, &datetime, nil)
// 	})
// }

func TestConvertString(t *testing.T) {
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 String")
		defer close(t)
		insertData(t, conn, "37")
		checkRows(t, conn, 1, "37")
	})
}

//	func TestConvertNullableString(t *testing.T) {
//		t.Run(t.Name(), func(t *testing.T) {
//			conn, close := setupTest(t, "col1 Nullable(String)")
//			defer close(t)
//			insertData(t, conn, "38", nil)
//			val := "38"
//			checkRows(t, conn, 2, &val, nil)
//		})
//	}
func TestConvertDecimal(t *testing.T) {
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 Decimal(15,3)")
		defer close(t)
		insertData(t, conn, decimal.New(39, 10))
		val, _ := decimal.New(39, 10).Float64()
		checkRows(t, conn, 1, val)
	})
}

//	func TestConvertNullableDecimal(t *testing.T) {
//		t.Run(t.Name(), func(t *testing.T) {
//			conn, close := setupTest(t, "col1 Nullable(Decimal(15,3))")
//			defer close(t)
//			insertData(t, conn, decimal.New(40, 10), nil)
//			val, _ := decimal.New(40, 10).Float64()
//			checkRows(t, conn, 2, &val, nil)
//		})
//	}
func TestTuple(t *testing.T) {
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 Tuple(int, varchar)")
		defer close(t)
		insertData(t, conn, "(41,'42')")
		checkRows(t, conn, 1, map[string]interface{}{"Field0": 41, "Field1": "42"})
	})
}

//	func TestArrayTuple(t *testing.T) {
//		t.Run(t.Name(), func(t *testing.T) {
//			conn, close := setupTest(t, "col1 Array(Tuple(s String, i Int32))")
//			defer close(t)
//			val := []map[string]interface{}{{"s": "43", "i": int32(43)}}
//			insertData(t, conn, val)
//			checkRows(t, conn, 1, val)
//		})
//	}
func TestArrayInt64(t *testing.T) {
	t.Run(t.Name(), func(t *testing.T) {
		conn, close := setupTest(t, "col1 ARRAY(INT64)")
		defer close(t)
		val := []int64{int64(45), int64(45)}
		insertData(t, conn, val)
		checkRows(t, conn, 1, val)
	})
}

//	func TestArrayNullableInt64(t *testing.T) {
//		t.Run(t.Name(), func(t *testing.T) {
//			conn, close := setupTest(t, "col1 Array(Nullable(Int64))")
//			defer close(t)
//			v := int64(45)
//			val := []*int64{&v, nil}
//			insertData(t, conn, val)
//			checkRows(t, conn, 1, val)
//		})
//	}

// todo: fixit
// func TestMap(t *testing.T) {
// 	t.Run(t.Name(), func(t *testing.T) {
// 		conn, close := setupTest(t, "col1 Map(UInt8, UInt8)")
// 		defer close(t)
// 		val := map[uint8]uint8{uint8(49): uint8(49)}
// 		insertData(t, conn, val)
// 		checkRows(t, conn, 1, val)
// 	})
// }
