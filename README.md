polaris-client
==============

Polaris-client is used for polaris service test. Written in Golang for high concurrency.


Build
=====

Dependency
----------
Using goes as the elasticsearch lib, so before build the source, please run the command:  
`go get github.com/belogik/goes`  
to download goes package for elasticsearch connection usage.

Build
-----
1. Add polaris-client to $GOPATH  
`export GOPATH=$GOPATH:<project-dir>`
2. Build  
`cd polaris-client/src`  
`go build -o <executable-name> main.go`

Functionalities
===============

Done
----
1. Upload files with multiple tasks and users
2. Upload dir with mutiple users (upload dir does not support multiple tasks)
3. List files with multiple users (dev branch)
4. Delete files(dir) (dev branch)
5. Delete all files in batch mode (dev branch)

TODO
----
1. Index document to ES
2. Delete document from ES
3. Create alias to ES
4. ...

Known Issues
============

1. If upload task number is larger than 300, pthread_create failure maybe thrown out @ubuntu 12.04 with file discriptor limitation as **1024**
2. Multiple users support still in dev branch, may be inefficient for real test. So needs enhancement.
