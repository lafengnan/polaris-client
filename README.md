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
`go build main.go`

Functionalities
===============

Done
----
1. Upload files with multiple tasks and users
2. Upload dir with mutiple users (upload dir does not support multiple tasks)
3. List files with multiple users

TODO
----
1. Delete files(dir)
2. Index document to ES
3. Delete document from ES
4. Create alias to ES
5. ...

Known Issues
============

1. If task number is lager than 300, pthread_create failure maybe thrown out @ubuntu 12.04 with file discriptor as **1024**
2. Multiple users support still in dev branch, may be inefficient for real test. So needs enhancement.
