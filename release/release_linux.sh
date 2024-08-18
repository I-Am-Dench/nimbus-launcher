#!/bin/bash

cd "$(dirname $0)"

rm -rf launcher
mkdir launcher

go build -tags release -o ./launcher/nimbus-launcher ..
cp ../LICENSE ./launcher

zip -r ../launcher-linux.zip ./launcher