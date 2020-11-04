#!/bin/bash
set -o errexit -o pipefail
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd ${DIR}
mkdir TEMP >& /dev/null || true
rm -rf TEMP/diff.out TEMP/mrtmp.pg345.txt
go run wc.go master pg345.txt sequential |& grep -vF rpc.Register: 
sort -n -k2 TEMP/mrtmp.pg345.txt | tail -20 | diff - mr-testout.txt > TEMP/diff.out || true
if [ -e TEMP/diff.out ]
then
if [ -s TEMP/diff.out ]
then
echo "Failed test. Output should be as in mr-testout.txt. Your output differs as follows (from diff.out):"
  cat TEMP/diff.out
else
  echo "Passed test"
fi
else
  echo "Test failed, no output detected"
fi

