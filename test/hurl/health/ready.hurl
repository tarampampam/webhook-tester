GET http://{{ host }}:{{ port }}/ready

HTTP/* 200

[Asserts]
bytes count == 0

# --- Head request should fails

HEAD http://{{ host }}:{{ port }}/ready

HTTP/* 404
