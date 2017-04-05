echo "Building..."
mkdir build
env GOOS=linux GOARCH=amd64 go build -o build/crawler . 

echo "Starting docker container..."
docker-compose up