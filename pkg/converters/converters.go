package converters

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
	"github.com/shopspring/decimal"
)

type Converter struct {
	convert    func(in interface{}) (interface{}, error)
	fieldType  data.FieldType
	matchRegex *regexp.Regexp
	scanType   reflect.Type
}

var matchRegexes = map[string]*regexp.Regexp{
	// for complex Arrays e.g. Array(Tuple)
	"Array()":                   regexp.MustCompile(`^Array\(.*\)`),
	"Date":                      regexp.MustCompile(`^Date\(?`),
	"Decimal":                   regexp.MustCompile(`^Decimal`),
	"Map()":                     regexp.MustCompile(`^Map\(.*\)`),
	"Nullable(Date)":            regexp.MustCompile(`^Nullable\(Date\(?`),
	"Nullable(Decimal)":         regexp.MustCompile(`^Nullable\(Decimal`),
	"Nullable(String)":          regexp.MustCompile(`Nullable\(String`),
	"SimpleAggregateFunction()": regexp.MustCompile(`^SimpleAggregateFunction\(.*\)`),
	"Tuple()":                   regexp.MustCompile(`^Tuple\(.*\)`),
}

var Converters = map[string]Converter{
	"Boolean": {
		fieldType: data.FieldTypeBool,
		scanType:  reflect.PtrTo(reflect.TypeOf(true)),
	},
	"Nullable(Boolean)": {
		fieldType: data.FieldTypeNullableBool,
		scanType:  reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(true))),
	},
	"Float64": {
		fieldType: data.FieldTypeFloat64,
		scanType:  reflect.PtrTo(reflect.TypeOf(float64(0))),
	},
	"Float32": {
		fieldType: data.FieldTypeFloat32,
		scanType:  reflect.PtrTo(reflect.TypeOf(float32(0))),
	},
	"Nullable(Float32)": {
		fieldType: data.FieldTypeNullableFloat32,
		scanType:  reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(float32(0)))),
	},
	"Nullable(Float64)": {
		fieldType: data.FieldTypeNullableFloat64,
		scanType:  reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(float64(0)))),
	},
	"Int64": {
		fieldType: data.FieldTypeInt64,
		scanType:  reflect.PtrTo(reflect.TypeOf(int64(0))),
	},
	"Int32": {
		fieldType: data.FieldTypeInt32,
		scanType:  reflect.PtrTo(reflect.TypeOf(int32(0))),
	},
	"Int16": {
		fieldType: data.FieldTypeInt16,
		scanType:  reflect.PtrTo(reflect.TypeOf(int16(0))),
	},
	"Int8": {
		fieldType: data.FieldTypeInt8,
		scanType:  reflect.PtrTo(reflect.TypeOf(int8(0))),
	},
	"UInt64": {
		fieldType: data.FieldTypeUint64,
		scanType:  reflect.PtrTo(reflect.TypeOf(uint64(0))),
	},
	"UInt32": {
		fieldType: data.FieldTypeUint32,
		scanType:  reflect.PtrTo(reflect.TypeOf(uint32(0))),
	},
	"UInt16": {
		fieldType: data.FieldTypeUint16,
		scanType:  reflect.PtrTo(reflect.TypeOf(uint16(0))),
	},
	"UInt8": {
		fieldType: data.FieldTypeUint8,
		scanType:  reflect.PtrTo(reflect.TypeOf(uint8(0))),
	},
	"Nullable(UInt64)": {
		fieldType: data.FieldTypeNullableUint64,
		scanType:  reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(uint64(0)))),
	},
	"Nullable(UInt32)": {
		fieldType: data.FieldTypeNullableUint32,
		scanType:  reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(uint32(0)))),
	},
	"Nullable(UInt16)": {
		fieldType: data.FieldTypeNullableUint16,
		scanType:  reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(uint16(0)))),
	},
	"Nullable(UInt8)": {
		fieldType: data.FieldTypeNullableUint8,
		scanType:  reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(uint8(0)))),
	},
	"Nullable(Int64)": {
		fieldType: data.FieldTypeNullableInt64,
		scanType:  reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(int64(0)))),
	},
	"Nullable(Int32)": {
		fieldType: data.FieldTypeNullableInt32,
		scanType:  reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(int32(0)))),
	},
	"Nullable(Int16)": {
		fieldType: data.FieldTypeNullableInt16,
		scanType:  reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(int16(0)))),
	},
	"Nullable(Int8)": {
		fieldType: data.FieldTypeNullableInt8,
		scanType:  reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(int8(0)))),
	},
	// covers DateTime with tz, DateTime64 - see regexes, Date32
	"Date": {
		fieldType:  data.FieldTypeTime,
		matchRegex: matchRegexes["Date"],
		scanType:   reflect.PtrTo(reflect.TypeOf(time.Time{})),
	},
	"DateTime": {
		fieldType: data.FieldTypeTime,
		scanType:  reflect.PtrTo(reflect.TypeOf(time.Time{})),
	},
	"DateTime64": {
		fieldType: data.FieldTypeTime,
		scanType:  reflect.PtrTo(reflect.TypeOf(time.Time{})),
	},
	"Timestamp": {
		fieldType: data.FieldTypeTime,
		scanType:  reflect.PtrTo(reflect.TypeOf(time.Time{})),
	},
	"Nullable(Date)": {
		fieldType:  data.FieldTypeNullableTime,
		matchRegex: matchRegexes["Nullable(Date)"],
		scanType:   reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(time.Time{}))),
	},
	"Nullable(String)": {
		fieldType:  data.FieldTypeNullableString,
		matchRegex: matchRegexes["Nullable(String)"],
		scanType:   reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(""))),
	},
	"Decimal": {
		convert:    decimalConvert,
		fieldType:  data.FieldTypeFloat64,
		matchRegex: matchRegexes["Decimal"],
		scanType:   reflect.PtrTo(reflect.TypeOf(decimal.Decimal{})),
	},
	"Nullable(Decimal)": {
		convert:    decimalNullConvert,
		fieldType:  data.FieldTypeNullableFloat64,
		matchRegex: matchRegexes["Nullable(Decimal)"],
		scanType:   reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(decimal.Decimal{}))),
	},
	"Tuple()": {
		convert:    jsonConverter,
		fieldType:  data.FieldTypeNullableJSON,
		matchRegex: matchRegexes["Tuple()"],
		scanType:   reflect.TypeOf((*interface{})(nil)).Elem(),
	},
	"Array()": {
		convert:    jsonConverter,
		fieldType:  data.FieldTypeNullableJSON,
		matchRegex: matchRegexes["Array()"],
		scanType:   reflect.TypeOf((*interface{})(nil)).Elem(),
	},
	"Map()": {
		convert:    jsonConverter,
		fieldType:  data.FieldTypeNullableJSON,
		matchRegex: matchRegexes["Map()"],
		scanType:   reflect.TypeOf((*interface{})(nil)).Elem(),
	},
	"String": {
		fieldType: data.FieldTypeString,
		scanType:  reflect.PtrTo(reflect.TypeOf("")),
	},
	"SimpleAggregateFunction()": {
		convert:    jsonConverter,
		fieldType:  data.FieldTypeNullableJSON,
		matchRegex: matchRegexes["SimpleAggregateFunction()"],
		scanType:   reflect.TypeOf((*interface{})(nil)).Elem(),
	},
}

var ComplexTypes = []string{"Map"}
var DatabendConverters = GetConverters()

func GetConverters() []sqlutil.Converter {
	var list []sqlutil.Converter
	for name, converter := range Converters {
		list = append(list, createConverter(name, converter))
	}
	return list
}

func GetConverter(columnType string) sqlutil.Converter {
	converter, ok := Converters[columnType]
	if ok {
		return createConverter(columnType, converter)
	}
	for name, converter := range Converters {
		if name == columnType {
			return createConverter(name, converter)
		}
		if converter.matchRegex != nil && converter.matchRegex.MatchString(columnType) {
			return createConverter(name, converter)
		}
	}
	return sqlutil.Converter{}
}

func createConverter(name string, converter Converter) sqlutil.Converter {
	convert := defaultConvert
	if converter.convert != nil {
		convert = converter.convert
	}
	return sqlutil.Converter{
		Name:           name,
		InputScanType:  converter.scanType,
		InputTypeRegex: converter.matchRegex,
		InputTypeName:  name,
		FrameConverter: sqlutil.FrameConverter{
			FieldType:     converter.fieldType,
			ConverterFunc: convert,
		},
	}
}

func jsonConverter(in interface{}) (interface{}, error) {
	if in == nil {
		return (*string)(nil), nil
	}
	jBytes, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var rawJSON json.RawMessage
	err = json.Unmarshal(jBytes, &rawJSON)
	if err != nil {
		return nil, err
	}
	return &rawJSON, nil
}

func defaultConvert(in interface{}) (interface{}, error) {
	if in == nil {
		return reflect.Zero(reflect.TypeOf(in)).Interface(), nil
	}
	return reflect.ValueOf(in).Elem().Interface(), nil
}

func decimalConvert(in interface{}) (interface{}, error) {
	if in == nil {
		return float64(0), nil
	}
	v, ok := in.(*decimal.Decimal)
	if !ok {
		return nil, fmt.Errorf("invalid decimal - %v", in)
	}
	f, _ := (*v).Float64()
	return f, nil
}

func decimalNullConvert(in interface{}) (interface{}, error) {
	if in == nil {
		return float64(0), nil
	}
	v, ok := in.(**decimal.Decimal)
	if !ok {
		return nil, fmt.Errorf("invalid decimal - %v", in)
	}
	if *v == nil {
		return (*float64)(nil), nil
	}
	f, _ := (*v).Float64()
	return &f, nil
}
