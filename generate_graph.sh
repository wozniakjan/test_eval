#!/bin/bash

export JOB_NAME=test_branch_origin_extended_builds 
export BUILD_ID=439 
./run.sh
echo Generating graph from out_${BUILD_ID}-${JOB_NAME}/stats.json to graph_$JOB_NAME.html
go run graph.go -i outs/${BUILD_ID}-${JOB_NAME}/stats.json -o graph_$JOB_NAME.html
