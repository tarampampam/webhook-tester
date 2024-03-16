GET http://{{ host }}:{{ port }}/c1eb8242-0cf8-47ec-927a-69eda4a989a8

HTTP 404

[Asserts]
header "Content-Type" contains "text/html"
body contains "WebHook: Not Found"
