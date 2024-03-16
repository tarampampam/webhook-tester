GET http://{{ host }}:{{ port }}/api/version

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.version" isString
