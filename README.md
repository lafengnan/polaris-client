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
`cd src`  
`go build -o <excutable_name> main.go`

Functionalities
===============

Done
----
1. Upload files with multiple tasks and users
2. Upload dir with mutiple users (upload dir does not support multiple tasks)
3. List files with multiple users
4. Delete files(dir)

TODO
----
1. Index document to ES
2. Delete document from ES
3. Create alias to ES
4. ...

Known Issues
============

1. If task number is lager than 300, pthread_create failure maybe thrown out @ubuntu 12.04 with file discriptor limitaion as **1024**
2. Multiple users support still in dev branch, may be inefficient for real test. So needs enhancement.
