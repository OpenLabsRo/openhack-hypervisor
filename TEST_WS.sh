#!/bin/bash

TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjE0ODQyNDUsInBhc3N3b3JkIjoiJDJhJDEyJC9EbHVwOVVpVUwwano5cnpHNWl3Y08vVnVnLzF0Q2pXWFdxZGN0dmVpem1CcFNONXlUVUY2IiwidXNlcm5hbWUiOiJzdW5zaGluZSJ9.oolxP1SjuKPHJrPeTduTh-XLJqbDpW9sW1sj7KAVneg"
ROUTE="wss://$1/hypervisor/ws/stages/$2/tests/$3?authorization=${TOKEN}"
echo $ROUTE

wscat --connect "$ROUTE"
