POST http://{{ host }}:{{ port }}/api/session # create a session with all default values

HTTP 200

[Captures]
session_uuid: jsonpath "$.uuid"

# ---

GET http://{{ host }}:{{ port }}/ws/session/{{ session_uuid }}
Connection: Upgrade
Upgrade: websocket
Sec-WebSocket-Version: 13

HTTP 400

[Asserts]
header "Content-Type" contains "text/plain"
body contains "'Sec-WebSocket-Key' header is missing"

# --- Cleanup

DELETE http://{{ host }}:{{ port }}/api/session/{{ session_uuid }}

HTTP 200
