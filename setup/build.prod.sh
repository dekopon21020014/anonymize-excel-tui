#!/bin/bash
docker build --platform=linux/amd64 -f ./Dockerfile -t anonymize-csv:amd64 ../src
docker save -o anonymize-csv.tar anonymize-csv:amd64
#docker build --platform=linux/amd64 -f ./excel.dockerfile -t anonymize-excel:amd64 ../src
#docker save -o anonymize-excel.tar anonymize-excel:amd64
