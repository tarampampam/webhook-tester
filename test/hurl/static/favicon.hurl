GET http://{{ host }}:{{ port }}/favicon.ico
Accept: */*

HTTP 200

[Asserts]
header "Content-Type" contains "image"
bytes startsWith hex,00000100030030;

# --- Head request request should not return a body

HEAD http://{{ host }}:{{ port }}/favicon.ico
Accept: */*

HTTP 200

[Asserts]
bytes count == 0
