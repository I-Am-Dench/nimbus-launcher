#!/bin/bash

cd "$(dirname $0)"

rm -rf launcher
mkdir launcher

cp ../LICENSE ./launcher
cp ../README.md ./launcher

go build -tags release -o ./launcher/nimbus-launcher ..

zip -9 -r ../launcher-linux-amd64.zip ./launcher