#!/bin/sh
if [ -z $1 ];then
echo "please provide BACKUP file name"
exit 1
fi
echo "BACKUP file name:$1"
BACKUP_FILE="/var/lib/postgresql/data/$1"
echo "restoring BACKUP file $BACKUP_FILE"
sudo docker exec -it filesync-db psql -U filesync -d postgres -c "DROP DATABASE filesync"
sudo docker exec -it filesync-db psql -U filesync -f $BACKUP_FILE postgres