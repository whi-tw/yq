# Date Time

Various operators for parsing and manipulating dates. 

## format_date time
This uses the golangs built in time library for parsing and formatting date times.

`layout` specifies the current format of the datetime, and `format` is the target format.

`format_datetime(layout, format)`, or if you are using the standard RFC3339 layout (e.g. "2006-01-02T15:04:05Z07:00"), then you can simply call `format_datetime(format)`.

See https://pkg.go.dev/time#pkg-constants for examples on layouts and formats.
