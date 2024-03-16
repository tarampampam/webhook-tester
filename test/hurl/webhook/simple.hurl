# --- Create a session first

POST http://{{ host }}:{{ port }}/api/session
Content-Type: application/json

{
  "status_code": 201,
  "content_type": "text/plain",
  "response_delay": 0,
  "response_content_base64": "Zm9vIGJhcg=="
}

HTTP 200

[Captures]
session_uuid: jsonpath "$.uuid"

# --- Method GET

GET http://{{ host }}:{{ port }}/{{ session_uuid }}
Foo-Header: BarValue

HTTP 201

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method HEAD

HEAD http://{{ host }}:{{ port }}/{{ session_uuid }}
Foo-Header: BarValue

HTTP 201

[Asserts]
header "Content-Type" contains "text/plain"
bytes count == 0

# --- Method POST

POST http://{{ host }}:{{ port }}/{{ session_uuid }}
Foo-Header: BarValue

```
bar baz
```

HTTP 201

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method PUT

PUT http://{{ host }}:{{ port }}/{{ session_uuid }}
Foo-Header: BarValue

```
bar baz
```

HTTP 201

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method PATCH

PATCH http://{{ host }}:{{ port }}/{{ session_uuid }}
Foo-Header: BarValue

```
bar baz
```

HTTP 201

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method DELETE

DELETE http://{{ host }}:{{ port }}/{{ session_uuid }}
Foo-Header: BarValue

```
bar baz
```

HTTP 201

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method OPTIONS

OPTIONS http://{{ host }}:{{ port }}/{{ session_uuid }}
Foo-Header: BarValue

```
bar baz
```

HTTP 201

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method TRACE

TRACE http://{{ host }}:{{ port }}/{{ session_uuid }}
Foo-Header: BarValue

```
bar baz
```

HTTP 201

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Check the recorded requests

GET http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}/requests

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$" count == 8 # GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS, TRACE
jsonpath "$[*].content_base64" includes "YmFyIGJhego="
jsonpath "$[*].headers[*].name" includes "Foo-Header"
jsonpath "$[*].headers[*].value" includes "BarValue"
jsonpath "$[0].url" == "/{{ session_uuid }}"

# --- Cleanup

DELETE http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}

HTTP 200
