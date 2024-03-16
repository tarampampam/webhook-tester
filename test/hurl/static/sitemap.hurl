GET http://{{ host }}:{{ port }}/sitemap.xml

HTTP 200

[Asserts]
header "Content-Type" contains "text/xml"

# --- Head request request should not return a body

HEAD http://{{ host }}:{{ port }}/sitemap.xml
Accept: */*

HTTP 200

[Asserts]
bytes count == 0
