#!/bin/bash

TESTARGS=$@
UDS="/tmp/e2e-csi-sanity.sock"
CSI_ENDPOINTS="127.0.0.1:9998"
CSI_ENDPOINTS="$CSI_ENDPOINTS unix://${UDS}"
CSI_ENDPOINTS="$CSI_ENDPOINTS ${UDS}"
CSI_MOCK_VERSION="master"

#
# $1 - endpoint for mock.
# $2 - endpoint for csi-sanity in Grpc format.
#      See https://github.com/grpc/grpc/blob/master/doc/naming.md
runTest()
{
	CSI_ENDPOINT=$1 mock &
	local pid=$!

	csi-sanity $TESTARGS --csi.endpoint=$2; ret=$?
	kill -9 $pid

	if [ $ret -ne 0 ] ; then
		exit $ret
	fi
}

#
# TODO Once 0.2.0 change gets merged into gocsi repo, switch back to "go get"
#
git clone https://github.com/thecodeteam/gocsi $GOPATH/src/github.com/thecodeteam/gocsi
pushd $GOPATH/src/github.com/thecodeteam/gocsi
git checkout $CSI_MOCK_VERSION
make build
popd 
#
#go get -u github.com/thecodeteam/gocsi/mock
cd cmd/csi-sanity
  make clean install || exit 1
cd ../..

runTest "tcp://127.0.0.1:9998" "127.0.0.1:9998"
rm -f $UDS
runTest "unix://${UDS}" "unix://${UDS}"
rm -f $UDS
runTest "${UDS}" "${UDS}"
rm -f $UDS

exit 0
