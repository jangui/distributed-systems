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
CLUSTERCOUNT=7
ITERATIONCOUNT=8
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd ${DIR}
mkdir TEMP >& /dev/null || true
rm -rf TEMP/mrtmp.points.txt*
rm -rf TEMP/worker.*.socket
rm -rf TEMP/master.socket

go run kmeans.go master points.txt sequential ${CLUSTERCOUNT} ${ITERATIONCOUNT}|& grep -vF rpc.Register: &

MASTERPID=$!
SUBPROCS=${MASTERPID}
sleep 1 # wait for master to be able to accept requests
for WORKER in $(seq $WORKERCOUNT); do 
    go run kmeans.go worker TEMP/master.socket TEMP/worker.${WORKER}.socket ${CLUSTERCOUNT} ${ITERATIONCOUNT}&
    SUBPROCS="${SUBPROCS} $!"
done

wait $MASTERPID
if [ -e TEMP/mrtmp.points.txt ]; then
  ./points.py collect --input TEMP/mrtmp.points.txt-[0-9]-[0-9]
else
  echo "Test failed, no output detected"
fi

