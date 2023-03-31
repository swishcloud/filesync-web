#/bin/sh
set -e
IMAGE_TAG=filesync-web:latest
if [ -d ".dist" ];then
    rm .dist -r
fi
mkdir .dist
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o ./.dist/app
cp templates  ./.dist -r
cp static  ./.dist -r
cp migrations  ./.dist -r
sudo docker build --tag $IMAGE_TAG -f docker/dockerfile ./.dist