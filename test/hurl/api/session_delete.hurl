POST http://{{ host }}:{{ port }}/api/session # create a new session

HTTP 200

[Captures]
session_uuid: jsonpath "$.uuid"

# --- Should be OK

DELETE http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.success" == true

# --- Should fails

DELETE http://{{ host }}:{{ port }}/api/session/{{ session_uuid }} # repeat

HTTP 404

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.code" == 404
jsonpath "$.message" contains "not found"
