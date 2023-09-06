# Databend data source for Grafana

This project was forked from [grafana/clickhouse-datasource](https://github.com/grafana/clickhouse-datasource), and is based on clickhouse-datasource version 3.3.0, with adaptations made for databend. Thanks to grafana/clickhouse-datasource for the original project.


<img src="https://github.com/cadl/grafana-databend-datasource/assets/1629582/05cd5a1e-e0bf-420b-88ad-4204913f8eed" width=200>
<img src="https://github.com/cadl/grafana-databend-datasource/assets/1629582/191da5c9-0805-4ca3-b63a-eb3e71b8e97c" width=200>
<img src="https://github.com/cadl/grafana-databend-datasource/assets/1629582/5c99cdfc-9fc0-45ba-ab2d-6822e647f19c" width=200>

## Installation

For detailed instructions on how to install the plugin on Grafana Cloud or
locally, please checkout the [Plugin installation docs](https://grafana.com/docs/grafana/latest/plugins/installation/).


## Configuration

### Manual configuration

Once the plugin is installed on your Grafana instance, follow [these
instructions](https://grafana.com/docs/grafana/latest/datasources/add-a-data-source/)
to add a new data source, and enter configuration options.

## Building queries

The query editor allows you to query Databend to return time series or
tabular data. Queries can contain macros which simplify syntax and allow for
dynamic parts.

### Time series

Time series visualization options are selectable after adding a `datetime`
field type to your query. This field will be used as the timestamp. You can
select time series visualizations using the visualization options. Grafana
interprets timestamp rows without explicit time zone as UTC. Any column except
`time` is treated as a value column.

#### Multi-line time series

To create multi-line time series, the query must return at least 3 fields in
the following order:
- field 1:  `datetime` field with an alias of `time`
- field 2:  value to group by
- field 3+: the metric values

For example:
```sql
SELECT log_time AS time, machine_group, avg(disk_free) AS avg_disk_free
FROM mgbench.logs1
GROUP BY machine_group, log_time
ORDER BY log_time
```

### Tables

Table visualizations will always be available for any valid Databend query.

### Visualizing logs with the Logs Panel

To use the Logs panel your query must return a timestamp and string values. To default to the logs visualization in Explore mode, set the timestamp alias to *log_time*.

For example:
```sql
SELECT log_time AS log_time, machine_group, toString(avg(disk_free)) AS avg_disk_free
FROM logs1
GROUP BY machine_group, log_time
ORDER BY log_time
```

To force rendering as logs, in absence of a `log_time` column, set the Format to `Logs` (available from 2.2.0).

### Macros

To simplify syntax and to allow for dynamic parts, like date range filters, the query can contain macros.

Here is an example of a query with a macro that will use Grafana's time filter:
```sql
SELECT date_time, data_stuff
FROM test_data
WHERE $__timeFilter(date_time)
```

| Macro                                        | Description                                                                                                                                                                         | Output example                                                        |
|----------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------|
| *$__timeFilter(columnName)*                  | Replaced by a conditional that filters the data (using the provided column) based on the time range of the panel in seconds                                                         | `time >= '1480001790' AND time <= '1482576232' )`                     |
| *$__dateFilter(columnName)*                  | Replaced by a conditional that filters the data (using the provided column) based on the date range of the panel                                                                    | `date >= '2022-10-21' AND date <= '2022-10-23' )`                     |
| *$__timeFilter_ms(columnName)*               | Replaced by a conditional that filters the data (using the provided column) based on the time range of the panel in milliseconds                                                    | `time >= '1480001790671' AND time <= '1482576232479' )`               |
| *$__fromTime*                                | Replaced by the starting time of the range of the panel casted to DateTime                                                                                                          | `toDateTime(intDiv(1415792726371,1000))`                              |
| *$__toTime*                                  | Replaced by the ending time of the range of the panel casted to DateTime                                                                                                            | `toDateTime(intDiv(1415792726371,1000))`                              |
| *$__interval_s*                              | Replaced by the interval in seconds                                                                                                                                                 | `20`                                                                  |
| *$__timeInterval(columnName)*                | Replaced by a function calculating the interval based on window size in seconds, useful when grouping                                                                               | `toStartOfInterval(toDateTime(column), INTERVAL 20 second)`           |
| *$__timeInterval_ms(columnName)*             | Replaced by a function calculating the interval based on window size in milliseconds, useful when grouping                                                                          | `toStartOfInterval(toDateTime64(column, 3), INTERVAL 20 millisecond)` |
| *$__conditionalAll(condition, $templateVar)* | Replaced by the first parameter when the template variable in the second parameter does not select every value. Replaced by the 1=1 when the template variable selects every value. | `condition` or `1=1`                                                  |

The plugin also supports notation using braces {}. Use this notation when queries are needed inside parameters.


### Templates and variables

To add a new query variable, refer to [Add a query
variable](https://grafana.com/docs/grafana/latest/variables/variable-types/add-query-variable/).

After creating a variable, you can use it in your queries by using
[Variable syntax](https://grafana.com/docs/grafana/latest/variables/syntax/).
For more information about variables, refer to [Templates and
variables](https://grafana.com/docs/grafana/latest/variables/).

### Ad Hoc Filters

Ad hoc filters allow you to add key/value filters that are automatically added
to all metric queries that use the specified data source, without being
explicitly used in queries.

By default, Ad Hoc filters will be populated with all Tables and Columns.  If
you have a default database defined in the Datasource settings, all Tables from
that database will be used to populate the filters. As this could be
slow/expensive, you can introduce a second variable to allow limiting the
Ad Hoc filters. It should be a `constant` type named `databend_adhoc_query`
and can contain: a comma delimited list of databases, just one database, or a
database.table combination to show only columns for a single table.

For more information on Ad Hoc filters, check the [Grafana
docs](https://grafana.com/docs/grafana/latest/variables/variable-types/add-ad-hoc-filters/)

#### Using a query for Ad Hoc filters

The second `databend_adhoc_query` also allows any valid query. The
query results will be used to populate your ad-hoc filter's selectable filters.
You may choose to hide this variable from view as it serves no further purpose.

For example, if `databend_adhoc_query` is set to `SELECT DISTINCT
machine_name FROM mgbench.logs1` you would be able to select which machine
names are filtered for in the dashboard.

## Learn more

* Add [Annotations](https://grafana.com/docs/grafana/latest/dashboards/annotations/).
* Configure and use [Templates and variables](https://grafana.com/docs/grafana/latest/variables/).
* Add [Transformations](https://grafana.com/docs/grafana/latest/panels/transformations/).
* Set up alerting; refer to [Alerts overview](https://grafana.com/docs/grafana/latest/alerting/).
