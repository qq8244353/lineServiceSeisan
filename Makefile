build:
	GOOS=linux GOARCH=amd64 go build -o hello main.go
zip:
	zip hello.zip hello
