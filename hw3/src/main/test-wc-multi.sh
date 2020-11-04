#!/bin/bash

SUBPROCS=""
function finish {
    SUBPROCS=$(ps -o pid= --ppid $$)
    for SUBPROC in ${SUBPROCS}; do
        pkill -P ${SUBPROC} >& /dev/null || true 
    done
}

trap '' TERM
trap finish EXIT

WORKERCOUNT=3

while [[ "$#" -gt 0 ]]
do
key="$1"
case "$key" in
    --workers)
        shift
        if [ "$1" -eq "$1" -o "$1" -lt "1" ] 2>/dev/null
        then
            WORKERCOUNT="$1"
        else
            echo "Expected positive integer, but got $1"
            exit 1
        fi
        ;;
    *)
        echo "Synax error. Supported flags:"
        echo "    $0 --workers [workercount]"
        exit 1
esac
shift
done

set -o errexit -o pipefail
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd ${DIR}
mkdir TEMP >& /dev/null || true
rm -rf TEMP/diff.out TEMP/mrtmp.pg345.txt
rm -rf TEMP/worker.*.socket
rm -rf TEMP/master.socket
go run wc.go master pg345.txt TEMP/master.socket |& grep -vF rpc.Register: &
MASTERPID=$!
SUBPROCS=${MASTERPID}
sleep 1 # wait for master to be able to accept requests
for WORKER in $(seq $WORKERCOUNT); do 
    go run wc.go worker TEMP/master.socket TEMP/worker.${WORKER}.socket &
    SUBPROCS="${SUBPROCS} $!"
done

wait $MASTERPID
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

