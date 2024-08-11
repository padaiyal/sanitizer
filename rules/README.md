# Sanitization rules

The rules contain the patterns/specific fields and the approach to sanitize them.

## Rule file
Each rule file is in the yaml format stored as `rules/<file_extension>.yaml`.<br>
For example, the rule file for HARs will be `rules/har.yaml`
Each file can have one or more rules.

### File format
The format is as follows:
```
description: <Description of the file format and some info on the find of info sanitized.>
format: <json>
rules:
    <json_path_pattern>:
        description: <Information on what this rule sanitizes>
        action: <contextual_replacement|remove>
    ...
```
For an actual rule file, refer to [har.yaml](har.yaml)

### Rule format
As shown in the file format example above, a rule format looks like this:
```
<json_path_pattern>:
    description: <Information on what this rule sanitizes>
    action: <contextual_replacement|remove>
```
The `rules` section can contain one or more of these.
The `action` for each rule can be one of the following:
 - `contextual_replacement` - If this is chosen, during the sanitization of this file, the identical values are replaced with the same replacement value for context preservation. For example, there may be multiple rules sanitizing multiple fields with the sensitive value `topsecret`, and in this action it replaces all occurrences of `topsecret` with the same value.
 - `remove` - Replaces the sensitive value with `<REMOVED>`.