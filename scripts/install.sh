set -e
if [ ! -d .cache ];then
mkdir .cache
fi
#set up and run database
sudo  docker compose -p filesync-web-project -f docker-compose-postgres.yaml up -d
#generate TLS certificate
openssl req -newkey rsa:4096 \
-x509 \
-sha256 \
-days 365 \
-nodes \
-out .cache/localhost.crt \
-keyout .cache/localhost.key \
-subj "/C=CH/ST=GUANGDNG/L=SHENZHEN/O=SECURITY/OU=IT DEPARTMENT/CN=localhost"
# build docker image
docker/build.sh
#migration
sudo docker run filesync-web:latest migrate sql --conn_info "postgres://filesync:secret@192.168.1.1:6010/filesync?sslmode=disable"