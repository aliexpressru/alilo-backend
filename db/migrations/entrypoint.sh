#!/bin/bash

set -e

DBSTRING="host=$DBHOST user=$DBUSER password=$DBPASSWORD dbname=$DBNAME sslmode=$DBSSL"

echo "Running goose migrations..."
goose postgres "$DBSTRING" up

