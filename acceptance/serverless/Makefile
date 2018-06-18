build:
	go get -u github.com/iopipe/iopipe-go
	dep ensure
	env GOOS=linux go build -ldflags="-s -w" -o bin/success success/main.go