GET http://{{ host }}:{{ port }}/foobar404

HTTP 404

[Asserts]
header "Content-Type" contains "text/html"
xpath "string(/html/head/title)" contains "Not found" # Check title

# --- Head request request should not return a body

HEAD http://{{ host }}:{{ port }}/foobar404
Accept: */*

HTTP 404

[Asserts]
bytes count == 0

# --- Simple (not existent) file requested

GET http://{{ host }}:{{ port }}/foo/bar/404.json

HTTP 404

[Asserts]
header "Content-Type" contains "text/html"
xpath "string(/html/head/title)" contains "Not found" # Check title
