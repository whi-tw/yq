# Date Time

Various operators for parsing and manipulating dates. 

## format_date time
This uses the golangs built in time library for parsing and formatting date times.

`layout` specifies the current format of the datetime, and `format` is the target format.

`format_datetime(layout, format)`, or if you are using the standard RFC3339 layout (e.g. "2006-01-02T15:04:05Z07:00"), then you can simply call `format_datetime(format)`.

See https://pkg.go.dev/time#pkg-constants for examples on layouts and formats.

{% hint style="warning" %}
Note that versions prior to 4.18 require the 'eval/e' command to be specified.&#x20;

`yq e <exp> <file>`
{% endhint %}

## Format: from ISO standard date time
Providing a single parameter assumes a standard ISO datetime format. If the target format is not a valid yaml datetime format, the result will be a string tagged node.

Given a sample.yml file of:
```yaml
a: 2001-12-15T02:59:43.1Z
```
then
```bash
yq '.a |= format_datetime("Monday, 02-Jan-06 at 3:04PM")' sample.yml
```
will output
```yaml
a: Saturday, 15-Dec-01 at 2:59AM
```

## Format: from custom date time
if the target format is a valid yaml datetime format, then it will automatically get the !!timestamp tag.

Given a sample.yml file of:
```yaml
a: Saturday, 15-Dec-01 at 2:59AM
```
then
```bash
yq '.a |= format_datetime("Monday, 02-Jan-06 at 3:04PM"; "2006-01-02")' sample.yml
```
will output
```yaml
a: 2001-12-15
```

## Now
Given a sample.yml file of:
```yaml
a: cool
```
then
```bash
yq '.updated = now' sample.yml
```
will output
```yaml
a: cool
updated: 2021-05-19T01:02:03Z
```

