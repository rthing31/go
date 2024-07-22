FUNCTION_NAME=hello-world-go-lambda

build:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap *.go
	zip lambda-handler.zip bootstrap

update_lambda: build
	aws lambda update-function-code --function-name $(FUNCTION_NAME) --zip-file fileb://lambda-handler.zip > /dev/null