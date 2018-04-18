#! /bin/sh

RETRIES=15

until psql -h localhost -U postgres -c "select 1;" > /dev/null 2>&1; do
  if [ $RETRIES -eq 0 ]; then
    exit 1
  fi

  sleep 0.3
  RETRIES=$(($RETRIES-1))
done

exit 0
