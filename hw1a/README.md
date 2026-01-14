# Go RESTful API Assignment

## Overview
A simple RESTful API built with Go and Gin framework for managing albums.

## Local Deployment

### Running locally
```
cd hw1a
go mod init example/web-service-gin
go get .
go run .
```

### Testing locally
```
# Get all albums
curl http://localhost:8080/albums

# Get album by ID
curl http://localhost:8080/albums/1

# Add new album
curl -X POST http://localhost:8080/albums \
  -H "Content-Type: application/json" \
  -d '{"id":"4","title":"New Album","artist":"Artist","price":39.99}'
```

## GCP Deployment

### Cloud Shell Deployment
1. Opened Google Cloud Shell Editor
2. Created project in `web-service-gin` directory
3. Initialized Go module: `go mod init example/web-service-gin`
4. Downloaded dependencies: `go get .`
5. Ran `go run .` in terminal
6. Server started on port 8080
7. Tested all endpoints using curl commands

### Screenshots
- [Screenshot 1: Server running and handling requests](./screenshots/1.png)
- [Screenshot 2: Testing GET /albums endpoint](./screenshots/2.png)
- [Screenshot 3: Testing POST /albums endpoint](./screenshots/3.png)
