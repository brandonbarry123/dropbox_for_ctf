#!/bin/bash  
sqlite3 dropbox.db "delete from userdata"
sqlite3 dropbox.db "delete from filedata"
sqlite3 dropbox.db "delete from sharedata"

rm -r userfs
rm -r filestore
mkdir userfs
mkdir filestore

rm filecount.txt
echo "1" > filecount.txt

echo 
