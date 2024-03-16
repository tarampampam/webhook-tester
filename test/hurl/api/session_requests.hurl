POST http://{{ host }}:{{ port }}/api/session # create a session
Content-Type: application/json

{
  "status_code": 201,
  "content_type": "application1/json2",
  "response_content_base64": "Zm9vIGJhcg=="
}

HTTP 200

[Captures]
session_uuid: jsonpath "$.uuid"

# --- Check the initial state

GET http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}/requests

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$" count == 0

# --- Send a simple request

POST http://{{ host }}:{{ port }}/{{ session_uuid }}/foobar
Foo-Header: BarValue

```
bar baz
```

HTTP 201

[Asserts]
header "Content-Type" contains "application1/json2"
body == "foo bar"

# --- Check the recorded requests again

GET http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}/requests

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$" count == 1
jsonpath "$[0].client_address" isString
jsonpath "$[0].content_base64" == "YmFyIGJhego="
jsonpath "$[0].created_at_unix" > 1600000000
jsonpath "$[0].headers[*].name" includes "Foo-Header"
jsonpath "$[0].headers[*].value" includes "BarValue"
jsonpath "$[0].method" == "POST"
jsonpath "$[0].url" == "/{{ session_uuid }}/foobar"

# --- Delete all requests

DELETE http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}/requests

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.success" == true

# --- Recorded requests should become empty

GET http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}/requests

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$" count == 0

# --- Cleanup

DELETE http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}

HTTP 200

# -- After the session removal, the request should fail

GET http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}/requests

HTTP 404
