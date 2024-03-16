POST http://{{ host }}:{{ port }}/api/session # create a session with all default values

HTTP 200

[Captures]
session_uuid: jsonpath "$.uuid"

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.created_at_unix" >= 1600000000
jsonpath "$.response.code" == 200 # default value
jsonpath "$.response.content_base64" == "" # default value
jsonpath "$.response.content_type" == "text/plain" # default value
jsonpath "$.response.delay_sec" == 0 # default value
jsonpath "$.uuid" matches "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

# --- Cleanup

DELETE http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}

HTTP 200

# --- Create a session with customized values

POST http://{{ host }}:{{ port }}/api/session
Content-Type: application/json

{
  "status_code": 201,
  "content_type": "application1/json2",
  "response_delay": 2,
  "response_content_base64": "Zm9vIGJhcg=="
}

HTTP 200

[Captures]
session_uuid: jsonpath "$.uuid"

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.created_at_unix" >= 1600000000
jsonpath "$.response.code" == 201
jsonpath "$.response.content_base64" == "Zm9vIGJhcg=="
jsonpath "$.response.content_type" == "application1/json2"
jsonpath "$.response.delay_sec" == 2
jsonpath "$.uuid" matches "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

# --- Cleanup

DELETE http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}

HTTP 200

# --- Wrong status code (too small)

POST http://{{ host }}:{{ port }}/api/session
Content-Type: application/json

{
  "status_code": 99
}

HTTP 400

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.code" == 400
jsonpath "$.message" contains "wrong status code"

# --- Wrong status code (too large)

POST http://{{ host }}:{{ port }}/api/session
Content-Type: application/json

{
  "status_code": 531
}

HTTP 400

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.code" == 400
jsonpath "$.message" contains "wrong status code"

# --- Wrong status code (too large)

POST http://{{ host }}:{{ port }}/api/session
Content-Type: application/json

{
  "response_delay": 31
}

HTTP 400

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.code" == 400
jsonpath "$.message" contains "response delay is too much"
