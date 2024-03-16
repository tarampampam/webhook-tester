GET http://{{ host }}:{{ port }}/live

HTTP 200

[Asserts]
bytes count == 0

# --- Head request

HEAD http://{{ host }}:{{ port }}/live

HTTP 200

[Asserts]
bytes count == 0
