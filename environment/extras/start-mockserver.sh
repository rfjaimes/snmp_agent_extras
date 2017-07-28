#!/bin/bash

echo "Starting mockserver"
cmdStartMock="/opt/mockserver/run_mockserver.sh -logLevel INFO -serverPort 80"
$cmdStartMock &

URL="http://localhost:80/clear"

isDown=true
while $isDown; do 
  sleep 1
  echo "Validating mockserver start"
  status_code=$(curl -X PUT -H "Content-Type: application/xml" -o /dev/null --silent --head --write-out '%{http_code}\n' $URL)
  echo "Status code: $status_code"
  if [ $status_code -eq "202" ]; then
    isDown=false
  fi
done;
echo "Mockserver has started!"

URL_EXP="http://localhost:80/expectation"
echo "Configuring mockserver..."
for i in `find /tmp/mocks -type f -name '*.req' | sort`; do
  echo "\n"
  echo "Loading '$i'...";
  curl -X PUT -H "Content-Type: application/json; charset=utf-8" --data @$i $URL_EXP
  echo "\n"
done
echo "Mockserver configured"

tail -f /opt/mockserver/mockserver.log
