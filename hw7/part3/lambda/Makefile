build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go
	zip function.zip bootstrap
	rm bootstrap

clean:
	rm -f bootstrap function.zip