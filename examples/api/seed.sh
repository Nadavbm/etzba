#!/bin/sh

TOKEN=$( curl -X POST http://localhost:8080/v1/signup \
	        -H 'Content-Type: application/json' \
	        -d '{"name": "etzba","email": "etzba@etzba.com","password": "Pass1234"}' | jq '.token' )

echo session is $TOKEN | sed 's/"//g'

TKN=$( echo $TOKEN | sed 's/"//g')
echo TKN $TKN

PAYLOAD=$( jq -n \
                  --arg t $TKN \
                  '{ "apiAuth": { "method": "Bearer", "token": $t } }' )

echo payload $PAYLOAD

echo $PAYLOAD > secret.json
