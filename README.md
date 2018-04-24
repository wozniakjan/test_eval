# `test_eval`

Repo with a helper for identifying slower running parts of origin extended tests.

- `run.sh` - by default, fetches last successful build log from [an extended test](https://ci.openshift.redhat.com/jenkins/job/test_branch_origin_extended_builds)
- `top.go` - uses that build log and creates an output directory identifying slow windows in our tests and order them from slowest to fastest
- `graph.go` - generate html graph from `top` output

![graph example](/graph.png)

Example output may look like:
```
$ ls outs/423-test_branch_origin_extended_builds/
0001_1913.131_test_extended_builds_pipeline.go:437                                                       
0002_957.221_test_extended_builds_pipeline.go:201      
0003_516.175_test_extended_builds_image_source.go:82
0004_482.732_test_extended_builds_contextdir.go:101                                                      
0005_456.465_test_extended_builds_digest.go:65
...
```
Where name of the file means `[order]_[run time]_[name of the test file]:[line number]`, slowest tests have the lowest `order` number.

An excerpt from a log starts with the slowest identified parts of the test, called `window`
```
$ head out_423-test_branch_origin_extended_builds/0001_1913.131_test_extended_builds_pipeline.go:437
time: 1913.131

Window 0 - 552s
Apr  3 11:48:35.747: INFO: Running 'oc new-app --config=/tmp/extended-test-jenkins-pipeline-d45wr-gzvhd-user.kubeconfig --namespace=extended-test-jenkins-pipeline-d45wr-gzvhd -f /tmp/fixture-testdata-dir828470533/examples/jenkins/pipeline/maven-pipeline.yaml'
Apr  3 11:48:39.007: INFO: Running 'oc start-build --config=/tmp/extended-test-jenkins-pipeline-d45wr-gzvhd-user.kubeconfig --namespace=extended-test-jenkins-pipeline-d45wr-gzvhd openshift-jee-sample -o=name'
Apr  3 11:48:40.285: INFO: 
Apr  3 11:48:40.287: INFO: Waiting for openshift-jee-sample-1 to complete
Apr  3 11:57:41.804: INFO: Done waiting for openshift-jee-sample-1: util.BuildResult{BuildPath:"build/openshift-jee-sample-1", BuildName:"openshift-jee-sample-1", StartBuildStdErr:"", StartBuildStdOut:"build/openshift-jee-sample-1", StartBuildErr:error(nil), BuildConfigName:"", Build:(*build.Build)(0xc420e44f00), BuildAttempt:true, BuildSuccess:true, BuildFailure:false, BuildCancelled:false, BuildTimeout:false, LogDumper:(util.LogDumperFunc)(nil), Oc:(*util.CLI)(0xc421016c40)}
Apr  3 11:57:41.906: INFO: Running 'oc delete --config=/tmp/extended-test-jenkins-pipeline-d45wr-gzvhd-user.kubeconfig --namespace=extended-test-jenkins-pipeline-d45wr-gzvhd bc openshift-jee-sample'
Apr  3 11:57:42.545: INFO: Running 'oc delete --config=/tmp/extended-test-jenkins-pipeline-d45wr-gzvhd-user.kubeconfig --namespace=extended-test-jenkins-pipeline-d45wr-gzvhd bc openshift-jee-sample-docker'
```
And is followed by the entire log output of the test after the block of `windows`
