# mtest

> Functional testing of maya

Mtest will cater to all functional test needs of mayaserver. Eventually,
`mtest` or similar patterns may also be used to execute the functional tests 
of other openebs repositories. One may think of using `mtest` as part of CI
runs.

## Moving parts of mtest

- Will include a CLI
- Will include libraries e.g. aws-sdk-go to test APIs of a running Mayaserver
- Will provide options to install dependent programs prior to running a functional test-case
