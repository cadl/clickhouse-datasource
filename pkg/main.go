package main

import (
	"os"

	"github.com/cadl/grafana-databend-datasource/pkg/plugin"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/sqlds/v2"
)

func main() {
	if err := datasource.Manage("grafana-databend-datasource", newDatasource, datasource.ManageOpts{}); err != nil {
		log.DefaultLogger.Error(err.Error())
		os.Exit(1)
	}
}

func newDatasource(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	ds := sqlds.NewDatasource(&plugin.Databend{})
	return ds.NewDatasource(settings)
}
