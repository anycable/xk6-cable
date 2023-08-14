test-js:
	@k6 run -u 1 jslib/testSuite.js

OUTPUT ?= ./bin/k6

build:
	GOPRIVATE="go.k6.io/k6" xk6 build latest \
            --output $(OUTPUT) \
            --with github.com/anycable/xk6-anycable=.
