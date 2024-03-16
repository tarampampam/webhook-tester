# disabled until https://github.com/Orange-OpenSource/hurl/issues/2540 is not fixed

#GET http://{{ host }}:{{ port }}/metrics
#
#HTTP 200
#
#[Asserts]
#header "Content-Type" contains "text/plain"
#bytes count >= 50
#body contains "go_goroutines"

#
# --- Head request should fails
#
#HEAD http://{{ host }}:{{ port }}/metrics
#
#HTTP 404
