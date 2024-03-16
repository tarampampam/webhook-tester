GET http://{{ host }}:{{ port }}/api/settings

HTTP 200

[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.limits.max_requests" isInteger
jsonpath "$.limits.max_webhook_body_size" isInteger
jsonpath "$.limits.session_lifetime_sec" isInteger
