package macros_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cadl/grafana-databend-datasource/pkg/macros"
	"github.com/cadl/grafana-databend-datasource/pkg/plugin"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/sqlds/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DatabendDriver struct {
	sqlds.Driver
}

type MockDB struct {
	DatabendDriver
}

func (h *DatabendDriver) Macros() sqlds.Macros {
	var C = plugin.Databend{}

	return C.Macros()
}

func TestMacroFromTimeFilter(t *testing.T) {
	from, _ := time.Parse("2006-01-02T15:04:05.000Z", "2014-11-12T11:45:26.371Z")
	to, _ := time.Parse("2006-01-02T15:04:05.000Z", "2015-11-12T11:45:26.371Z")
	query := sqlds.Query{
		TimeRange: backend.TimeRange{
			From: from,
			To:   to,
		},
		RawSQL: "select foo from foo where bar > $__fromTime",
	}
	tests := []struct {
		want    string
		wantErr bool
		name    string
	}{
		{
			name: "should return timefilter",
			want: "TO_TIMESTAMP(1415792726)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := macros.FromTimeFilter(&query, []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("macroFromTimeFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMacroToTimeFilter(t *testing.T) {
	from, _ := time.Parse("2006-01-02T15:04:05.000Z", "2014-11-12T11:45:26.371Z")
	to, _ := time.Parse("2006-01-02T15:04:05.000Z", "2015-11-12T11:45:26.371Z")
	query := sqlds.Query{
		TimeRange: backend.TimeRange{
			From: from,
			To:   to,
		},
		RawSQL: "select foo from foo where bar > $__toTime",
	}
	tests := []struct {
		want    string
		wantErr bool
		name    string
	}{
		{
			name: "should return timefilter",
			want: "TO_TIMESTAMP(1447328726)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := macros.ToTimeFilter(&query, []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("macroToTimeFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMacroDateFilter(t *testing.T) {
	from, _ := time.Parse("2006-01-02T15:04:05.000Z", "2014-11-12T11:45:26.371Z")
	to, _ := time.Parse("2006-01-02T15:04:05.000Z", "2015-11-12T11:45:26.371Z")
	query := sqlds.Query{
		TimeRange: backend.TimeRange{
			From: from,
			To:   to,
		},
	}
	got, err := macros.DateFilter(&query, []string{"dateCol"})
	assert.Nil(t, err)
	assert.Equal(t, "dateCol >= '2014-11-12' AND dateCol <= '2015-11-12'", got)
}

func TestMacroTimeInterval(t *testing.T) {
	query := sqlds.Query{
		RawSQL:   "select $__timeInterval(col) from foo",
		Interval: time.Duration(20000000000),
	}
	got, err := macros.TimeInterval(&query, []string{"col"})
	assert.Nil(t, err)
	assert.Equal(t, "TO_TIMESTAMP( TO_UNIX_TIMESTAMP(TO_TIMESTAMP(col)) // 20 * 20)", got)
}

func TestMacroTimeIntervalMs(t *testing.T) {
	query := sqlds.Query{
		RawSQL:   "select $__timeInterval_ms(col) from foo",
		Interval: time.Duration(20000000000),
	}
	got, err := macros.TimeIntervalMs(&query, []string{"col"})
	assert.Nil(t, err)
	assert.Equal(t, "TO_TIMESTAMP( TO_UNIX_TIMESTAMP(TO_TIMESTAMP(col)) // 20 * 20)", got)
}

func TestMacroIntervalSeconds(t *testing.T) {
	query := sqlds.Query{
		RawSQL:   "select toStartOfInterval(col, INTERVAL $__interval_s second) AS time from foo",
		Interval: time.Duration(20000000000),
	}
	got, err := macros.IntervalSeconds(&query, []string{})
	assert.Nil(t, err)
	assert.Equal(t, "20", got)
}

func TestInterpolate(t *testing.T) {
	from, _ := time.Parse("2006-01-02T15:04:05.000Z", "2014-11-12T11:45:26.371Z")
	to, _ := time.Parse("2006-01-02T15:04:05.000Z", "2015-11-12T11:45:26.371Z")

	tableName := "my_table"
	tableColumn := "my_col"

	type test struct {
		name   string
		input  string
		output string
	}

	tests := []test{
		{input: "select * from foo where $__timeFilter(cast(sth as timestamp))", output: "select * from foo where cast(sth as timestamp) >= '1415792726' AND cast(sth as timestamp) <= '1447328726'", name: "databend timeFilter"},
		{input: "select * from foo where $__timeFilter(cast(sth as timestamp) )", output: "select * from foo where cast(sth as timestamp) >= '1415792726' AND cast(sth as timestamp) <= '1447328726'", name: "databend timeFilter with empty spaces"},
		{input: "select * from foo where ( date >= $__fromTime and date <= $__toTime ) limit 100", output: "select * from foo where ( date >= TO_TIMESTAMP(1415792726) and date <= TO_TIMESTAMP(1447328726) ) limit 100", name: "databend fromTime and toTime"},
		{input: "select * from foo where ( date >= $__fromTime ) and ( date <= $__toTime ) limit 100", output: "select * from foo where ( date >= TO_TIMESTAMP(1415792726) ) and ( date <= TO_TIMESTAMP(1447328726) ) limit 100", name: "databend fromTime and toTime inside a complex clauses"},
	}

	for i, tc := range tests {
		driver := MockDB{}
		t.Run(fmt.Sprintf("[%d/%d] %s", i+1, len(tests), tc.name), func(t *testing.T) {
			query := &sqlds.Query{
				RawSQL: tc.input,
				Table:  tableName,
				Column: tableColumn,
				TimeRange: backend.TimeRange{
					From: from,
					To:   to,
				},
			}
			interpolatedQuery, err := sqlds.Interpolate(&driver, query)
			require.Nil(t, err)
			assert.Equal(t, tc.output, interpolatedQuery)
		})
	}
}
