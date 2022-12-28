build:
	GOOS=linux GOARCH=amd64 go build -o hello ./cmd/handler/main.go
zip:
	zip hello.zip hello
clear:
	rm -rf hello hello.zip
