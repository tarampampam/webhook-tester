POST http://{{ host }}:{{ port }}/api/session # create a session
Content-Type: application/json

{
  "status_code": 529,
  "content_type": "application2/json1",
  "response_content_base64": "Zm9vIGJhcg=="
}

HTTP 200

[Captures]
session_uuid: jsonpath "$.uuid"

# --- Send a simple request

PUT http://{{ host }}:{{ port }}/{{ session_uuid }}/foo/bar.js?bla=1&bla2=baz
Foo-Header: BarValue

```
bar baz
```

HTTP 529

[Asserts]
header "Content-Type" contains "application2/json1"
body == "foo bar"

# --- Get all recorded requests

GET http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}/requests

HTTP 200

[Captures]
request_uuid: jsonpath "$[0].uuid"

# -- Get recorded request data

GET http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}/requests/{{ request_uuid }}

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.client_address" isString
jsonpath "$.content_base64" == "YmFyIGJhego="
jsonpath "$.created_at_unix" > 1600000000
jsonpath "$.headers[*].name" includes "Foo-Header"
jsonpath "$.headers[*].value" includes "BarValue"
jsonpath "$.method" == "PUT"
jsonpath "$.url" == "/{{ session_uuid }}/foo/bar.js?bla=1&bla2=baz"

# --- Delete the requests

DELETE http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}/requests/{{ request_uuid }}

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.success" == true

# --- The request should not exist anymore

GET http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}/requests/{{ request_uuid }}

HTTP 404

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.code" == 404
jsonpath "$.message" contains "not found"

# --- Cleanup

DELETE http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}

HTTP 200
