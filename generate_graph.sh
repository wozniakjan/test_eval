#!/bin/bash

export JENKINS=${JENKINS:-https://ci.openshift.redhat.com/jenkins}
#export JOB_NAME=test_branch_origin_extended_builds 
export JOB_NAME=test_branch_origin_extended_image_ecosystem
export BUILD_ID=$(curl -s $JENKINS/job/$JOB_NAME/api/json | jq '.lastSuccessfulBuild.number')
./run.sh
echo Generating graph from out/${BUILD_ID}-${JOB_NAME}/stats.json to graph.html
go run graph.go -i outs/${BUILD_ID}-${JOB_NAME}/stats.json -o graph.html
