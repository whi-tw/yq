
{% hint style="warning" %}
Note that versions prior to 4.18 require the 'eval/e' command to be specified.&#x20;

`yq e <exp> <file>`
{% endhint %}

## Format from ISO standard date time
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

## Format from custom date time
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

