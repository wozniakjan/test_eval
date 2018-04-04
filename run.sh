#!/bin/bash

JENKINS=https://ci.openshift.redhat.com/jenkins
JOB_NAME=test_branch_origin_extended_builds
#if left empty, finds last successfull build
BUILD_ID=

if [[ $BUILD_ID == "" ]]; then
    BUILD_ID=$(curl -s $JENKINS/job/$JOB_NAME/api/json | jq '.lastSuccessfulBuild.number')
fi
if [[ $BUILD_ID > 0 ]]; then
    LOG_FILE="${BUILD_ID}-${JOB_NAME}.log"
    curl -s https://ci.openshift.redhat.com/jenkins/job/$JOB_NAME/$BUILD_ID/consoleText > $LOG_FILE
else
    echo "invalid BUILD_ID $BUILD_ID"
    exit 1
fi

OUT=out_${BUILD_ID}-${JOB_NAME}
if [[ -d $OUT ]]; then
    rm -rf $OUT
fi

go run top.go -f $LOG_FILE -c -1 -o $OUT
