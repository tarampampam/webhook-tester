GET http://{{ host }}:{{ port }}/robots.txt

HTTP 200

[Asserts]
header "Content-Type" contains "text/plain"
bytes count >= 13

# --- Head request request should not return a body

HEAD http://{{ host }}:{{ port }}/robots.txt
Accept: */*

HTTP 200

[Asserts]
bytes count == 0
