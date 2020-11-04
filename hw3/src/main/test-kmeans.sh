#!/bin/bash
set -o errexit -o pipefail
CLUSTERCOUNT=7
ITERATIONCOUNT=8
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd ${DIR}
mkdir TEMP >& /dev/null || true
rm -rf TEMP/mrtmp.points.txt*
go run kmeans.go master points.txt sequential ${CLUSTERCOUNT} ${ITERATIONCOUNT}|& grep -vF rpc.Register: 
if [ -e TEMP/mrtmp.points.txt ]; then
  ./points.py collect --input TEMP/mrtmp.points.txt-[0-9]-[0-9]
else
  echo "Test failed, no output detected"
fi

