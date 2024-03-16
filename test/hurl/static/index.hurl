GET http://{{ host }}:{{ port }}/
User-Agent: Mozilla/5.0
Accept: */*
Accept-Language: en-US,en;q=0.5
Accept-Encoding: gzip, deflate, br
Connection: keep-alive

HTTP 200

[Asserts]
header "Content-Type" contains "text/html"
xpath "string(/html/head/title)" contains "WebHook Tester" # Check title
xpath "//*[@id='app']" count == 1 # App mounting point exist

# --- Head request request should not return a body

HEAD http://{{ host }}:{{ port }}/
Accept: */*

HTTP 200

[Asserts]
bytes count == 0
