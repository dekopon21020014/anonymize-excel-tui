#!/bin/bash
docker build --platform=linux/amd64 -f ./Dockerfile -t anonymize-excel:amd64 ../src
docker save -o anonymize-excel.tar anonymize-excel:amd64
