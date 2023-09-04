package plugin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	godatabend "github.com/databendcloud/databend-go"

	"github.com/grafana/clickhouse-datasource/pkg/converters"
	"github.com/grafana/clickhouse-datasource/pkg/macros"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
	"github.com/grafana/sqlds/v2"
	"github.com/pkg/errors"
)

// Clickhouse defines how to connect to a Clickhouse datasource
// type Clickhouse struct{}

type Databend struct{}

func (d *Databend) Connect(config backend.DataSourceInstanceSettings, message json.RawMessage) (*sql.DB, error) {
	settings, err := LoadSettings(config)
	if err != nil {
		return nil, err
	}
	// var tlsConfig *tls.Config

	// if settings.TlsAuthWithCACert || settings.TlsClientAuth {
	// 	tlsConfig, err = getTLSConfig(settings)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// } else if settings.Secure {
	// 	tlsConfig = &tls.Config{
	// 		InsecureSkipVerify: settings.InsecureSkipVerify,
	// 	}
	// }
	t, err := strconv.Atoi(settings.Timeout)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("invalid timeout: %s", settings.Timeout))
	}
	qt, err := strconv.Atoi(settings.QueryTimeout)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("invalid query timeout: %s", settings.QueryTimeout))
	}
	// protocol := clickhouse.Native
	// if settings.Protocol == "http" {
	// 	protocol = clickhouse.HTTP
	// }
	// compression := clickhouse.CompressionLZ4
	// if protocol == clickhouse.HTTP {
	// 	compression = clickhouse.CompressionGZIP
	// }
	// customSettings := make(clickhouse.Settings)
	// if settings.CustomSettings != nil {
	// 	for _, setting := range settings.CustomSettings {
	// 		customSettings[setting.Setting] = setting.Value
	// 	}
	// }

	// cfg := mysql.Config{
	// 	User:                 settings.Username,
	// 	Passwd:               settings.Password,
	// 	Net:                  "tcp",
	// 	Addr:                 fmt.Sprintf("%s:%d", settings.Server, settings.Port),
	// 	DBName:               settings.DefaultDatabase,
	// 	Timeout:              time.Duration(t) * time.Second,
	// 	ReadTimeout:          time.Duration(qt) * time.Second,
	// 	AllowNativePasswords: true,
	// 	Collation:            "utf8mb4_general_ci",
	// 	ParseTime:            true,
	// 	Loc:                  time.Local,
	// 	Params: map[string]string{
	// 		"timezone": "'Asia/Shanghai'",
	// 	},
	// 	// TLS:         tlsConfig,
	// }

	// connector, err := mysql.NewConnector(&cfg)
	cfg := godatabend.Config{
		Host:         fmt.Sprintf("%s:%d", settings.Server, settings.Port),
		User:         settings.Username,
		Password:     settings.Password,
		Database:     settings.DefaultDatabase,
		SSLMode:      godatabend.SSL_MODE_DISABLE,
		Timeout:      time.Duration(t) * time.Second,
		WaitTimeSecs: int64(qt),
		Location:     time.Local,
	}

	db, err := sql.Open("databend", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(t)
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	chErr := make(chan error, 1)
	go func() {
		err = db.PingContext(ctx)
		chErr <- err
	}()

	select {
	case err := <-chErr:
		if err != nil {
			// sql ds will ping again and show error
			log.DefaultLogger.Error(err.Error())
			return db, nil
		}
	case <-time.After(timeout * time.Second):
		return db, errors.New("connection timed out")
	}

	return db, settings.isValid()
}

func (d *Databend) Converters() []sqlutil.Converter {
	// todo: replace converters
	return converters.ClickhouseConverters
}

func (d *Databend) Macros() sqlds.Macros {
	return map[string]sqlds.MacroFunc{
		"fromTime":        macros.FromTimeFilter,
		"toTime":          macros.ToTimeFilter,
		"timeFilter_ms":   macros.TimeFilterMs,
		"timeFilter":      macros.TimeFilter,
		"dateFilter":      macros.DateFilter,
		"timeInterval_ms": macros.TimeIntervalMs,
		"timeInterval":    macros.TimeInterval,
		"interval_s":      macros.IntervalSeconds,
	}
}

func (d *Databend) Settings(config backend.DataSourceInstanceSettings) sqlds.DriverSettings {
	settings, err := LoadSettings(config)
	timeout := 60
	if err == nil {
		t, err := strconv.Atoi(settings.QueryTimeout)
		if err == nil {
			timeout = t
		}
	}
	return sqlds.DriverSettings{
		Timeout: time.Second * time.Duration(timeout),
		FillMode: &data.FillMissing{
			Mode: data.FillModeNull,
		},
	}
}

func (d *Databend) MutateQuery(ctx context.Context, req backend.DataQuery) (context.Context, backend.DataQuery) {
	return ctx, req
}

type MapField struct {
	dfField *data.Field
	t       string
}

func newMapField(name string, labels data.Labels, v interface{}) (*MapField, error) {
	var newInitialValueSlice interface{}
	var t string

	switch ov := v.(type) {
	case float64:
		newInitialValueSlice = []*float64{&ov}
		t = "number"
	case string:
		newInitialValueSlice = []*string{&ov}
		t = "string"
	case bool:
		newInitialValueSlice = []*bool{&ov}
		t = "bool"
	case nil:
		newInitialValueSlice = []*string{nil}
		t = "string"
	default:
		return nil, errors.New(fmt.Sprintf("unknown type: %T", v))
	}

	return &MapField{
		dfField: data.NewField(name, labels, newInitialValueSlice),
		t:       t,
	}, nil
}

func (mf *MapField) convValue(value interface{}) interface{} {
	switch mf.t {
	case "number":
		v, ok := value.(float64)
		if !ok {
			return nil
		}
		return &v
	case "string":
		v, ok := value.(string)
		if !ok {
			return nil
		}
		return &v
	case "bool":
		v, ok := value.(bool)
		if !ok {
			return nil
		}
		return &v
	default:
		panic(fmt.Sprintf("unknown type: %s", mf.t))
	}
}

func (mf *MapField) append(value interface{}) error {
	if value == nil {
		mf.dfField.Append(nil)
		return nil
	} else {
		v := mf.convValue(value)
		if v == nil {
			mf.dfField.Append(nil)
			return nil
		}
		mf.dfField.Append(v)
		return nil
	}
}

func flattenDFKvField(field *data.Field) ([]*data.Field, error) {
	var newFields []*data.Field
	firstRowKv := make(map[string]interface{})
	firstRowRawMessage := field.At(0).(*json.RawMessage)
	if firstRowRawMessage == nil {
		return newFields, nil
	}

	err := json.Unmarshal(*firstRowRawMessage, &firstRowKv)
	if err != nil {
		return nil, err
	}

	mapFields := make(map[string]*MapField)

	for k, v := range firstRowKv {
		mapField, err := newMapField(fmt.Sprintf("%s['%s']", field.Name, k), field.Labels, v)
		if err != nil {
			return nil, err
		}
		mapFields[k] = mapField
	}

	for i := 1; i < field.Len(); i++ {
		val := field.At(i).(*json.RawMessage)
		if val == nil {
			for _, v := range mapFields {
				v.append(nil)
			}
		} else {
			var rowValues map[string]interface{}
			err := json.Unmarshal(*val, &rowValues)
			if err != nil {
				return nil, err
			}
			for kField, vField := range mapFields {
				vField.append(rowValues[kField])
			}
		}
	}
	for _, v := range mapFields {
		newFields = append(newFields, v.dfField)
	}
	return newFields, nil
}

func (d *Databend) MutateResponse(ctx context.Context, res data.Frames) (data.Frames, error) {
	newRes := make(data.Frames, 0, len(res))
	for _, frame := range res {
		if frame.Meta.PreferredVisualization == data.VisType(data.VisTypeLogs) {
			// newFrame := data.NewFrame(frame.Name)
			var newFields []*data.Field

			for _, field := range frame.Fields {
				newFields = append(newFields, field)
				if (field.Type() == data.FieldTypeNullableJSON || field.Type() == data.FieldTypeJSON) && field.Len() > 0 {
					flattenedFields, err := flattenDFKvField(field)
					if err != nil {
						return nil, err
					}
					newFields = append(newFields, flattenedFields...)
				}
			}
			newFrame := data.NewFrame(frame.Name, newFields...)
			newFrame.SetMeta(frame.Meta)
			newRes = append(newRes, newFrame)
		} else {
			newRes = append(newRes, frame)
		}
	}
	return newRes, nil
}

func CheckMinServerVersion(conn *sql.DB, major, minor, patch uint64) (bool, error) {
	var version struct {
		Major uint64
		Minor uint64
		Patch uint64
	}
	var res string
	if err := conn.QueryRow("SELECT version()").Scan(&res); err != nil {
		return false, err
	}
	for i, v := range strings.Split(res, ".") {
		switch i {
		case 0:
			version.Major, _ = strconv.ParseUint(v, 10, 64)
		case 1:
			version.Minor, _ = strconv.ParseUint(v, 10, 64)
		case 2:
			version.Patch, _ = strconv.ParseUint(v, 10, 64)
		}
	}
	if version.Major < major || (version.Major == major && version.Minor < minor) || (version.Major == major && version.Minor == minor && version.Patch < patch) {
		return false, nil
	}
	return true, nil
}
