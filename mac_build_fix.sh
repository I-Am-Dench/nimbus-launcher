#!/bin/bash

CGO_CFLAGS="-DNS_FORMAT_ARGUMENT(A)=" go build -o nimbus-launcher