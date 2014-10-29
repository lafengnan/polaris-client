polaris-client
==============

Polaris-client is used for polaris service test. Written in Golang for high concurrency.


Build
=====

Dependency
----------
Using goes as the elasticsearch lib, so before build the source, please run the command:  
`go get github.com/belogik/goes`   to download goes package

Build
-----
`go build main.go`


Known Issues
============

1. If task number is lager than 300, pthread_create failure maybe thrown out @ubuntu 12.04 with file discriptor as *1024*
2. Multiple users support still in dev branch, may be inefficient for real test. So needs enhancement.
