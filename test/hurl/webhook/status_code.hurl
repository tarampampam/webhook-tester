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

GET http://{{ host }}:{{ port }}/{{ session_uuid }}/210
Foo-Header: BarValue

HTTP 210

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method HEAD

HEAD http://{{ host }}:{{ port }}/{{ session_uuid }}/211
Foo-Header: BarValue

HTTP 211

[Asserts]
header "Content-Type" contains "text/plain"
bytes count == 0

# --- Method POST

POST http://{{ host }}:{{ port }}/{{ session_uuid }}/212
Foo-Header: BarValue

```
bar baz
```

HTTP 212

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method PUT

PUT http://{{ host }}:{{ port }}/{{ session_uuid }}/213
Foo-Header: BarValue

```
bar baz
```

HTTP 213

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method PATCH

PATCH http://{{ host }}:{{ port }}/{{ session_uuid }}/214
Foo-Header: BarValue

```
bar baz
```

HTTP 214

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method DELETE

DELETE http://{{ host }}:{{ port }}/{{ session_uuid }}/215
Foo-Header: BarValue

```
bar baz
```

HTTP 215

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method OPTIONS

OPTIONS http://{{ host }}:{{ port }}/{{ session_uuid }}/216
Foo-Header: BarValue

```
bar baz
```

HTTP 216

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Method TRACE

TRACE http://{{ host }}:{{ port }}/{{ session_uuid }}/217
Foo-Header: BarValue

```
bar baz
```

HTTP 217

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- The final slash is a mistake

GET http://{{ host }}:{{ port }}/{{ session_uuid }}/210/ # <-- final slash

HTTP 201 # <-- 201 instead of 210

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Anything after the code - too

GET http://{{ host }}:{{ port }}/{{ session_uuid }}/210/foobar

HTTP 201 # <-- 201 instead of 210

[Asserts]
header "Content-Type" contains "text/plain"
body == "foo bar"

# --- Check the recorded requests

GET http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}/requests

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$" count == 10 # GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS, TRACE; + GET + GET
jsonpath "$[*].content_base64" includes "YmFyIGJhego="
jsonpath "$[*].headers[*].name" includes "Foo-Header"
jsonpath "$[*].headers[*].value" includes "BarValue"

# --- Cleanup

DELETE http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}

HTTP 200
