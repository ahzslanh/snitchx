.PHONY: build test lint demo demo-build demo-run clean

build:
	go build -o snitchx .

test:
	go test ./...

lint:
	golangci-lint run

demo: demo-build demo-run

demo-build:
	docker build -f demo/Dockerfile -t snitchx-demo .

demo-run:
	docker run --rm -v $(PWD)/demo:/output snitchx-demo

clean:
	rm -f snitchx
	rm -f demo/demo.gif

