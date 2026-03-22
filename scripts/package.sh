#/bin/sh
set -e
GIT_REV="$(git rev-parse --short HEAD)"
FILENAME="filesync-web-$GIT_REV.tar.gz"

if [ -d ".dist" ];then
    rm .dist -r
fi
mkdir .dist

if [ -d "$FILENAME" ];then
    rm "$FILENAMA"
fi

https_proxy=http://192.168.29.9:8081 CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o ./.dist/app
cp templates  ./.dist -r
cp static  ./.dist -r
cp migrations  ./.dist -r

tar -cvzf "$FILENAME" -C ./.dist .

echo build successful:
echo $(pwd)/$FILENAME