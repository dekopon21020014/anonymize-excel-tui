#!/bin/bash
docker build -f ./Dockerfile -t anonymize-excel:arm64 ../src
#docker save -o anonymize-excel.tar anonymize-excel:arm64
