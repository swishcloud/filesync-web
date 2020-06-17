mkdir .cache
#set up and run database
docker-compose -p filesync-web-project -f docker-compose-postgres.yaml up -d
#generate TLS certificate
openssl req -newkey rsa:4096 \
-x509 \
-sha256 \
-days 365 \
-nodes \
-out .cache/localhost.crt \
-keyout .cache/localhost.key \
-subj "/C=CH/ST=GUANGDNG/L=SHENZHEN/O=SECURITY/OU=IT DEPARTMENT/CN=localhost"
#migration
filesync-web migrate sql --conn_info "postgres://filesync:secret@localhost:6010/filesync?sslmode=disable"