mkdir .cache
docker-compose -p filesync-web-project -f docker-compose-postgres.yaml up -d
openssl req -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -out .cache/localhost.crt -keyout .cache/localhost.key -subj "/C=CH/ST=GUANGDNG/L=SHENZHEN/O=SECURITY/OU=IT DEPARTMENT/CN=localhost"