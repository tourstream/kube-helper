[![Travis-CI][travis-image]][travis-url]

[![Coverage Status][coveralls-image]][coveralls-url]

[![License][license-image]][license-url]

***

# Kube Helper

This tool is used in our workflow to help during the setup of a kubernetes deployment.

## Installation

1. Install go (https://golang.org/doc/install#install or http://www.golangbootcamp.com/book/get_setup) 
2. Run `go env` and check the `GOPATH` variable
3. Go into the `GOPATH` directory and execute `git clone https://github.com/tourstream/kube-helper.git`
4. Run `go get -u github.com/golang/dep`
5. Run `go install github.com/golang/dep/cmd/dep`
6. Run `dep ensure` which ensures you'll have all the dependencies
7. Go into the `kube-helper` directory and run `go build`

[coveralls-image]: https://coveralls.io/repos/github/tourstream/kube-helper/badge.svg
[coveralls-url]: https://coveralls.io/github/tourstream/kube-helper

[travis-image]: https://travis-ci.org/tourstream/typo3-redis-lock-strategy.svg?branch=master
[travis-url]: https://travis-ci.org/tourstream/typo3-redis-lock-strategy

[license-image]: https://img.shields.io/github/license/tourstream/kube-helper.svg?style=flat-square
[license-url]: https://github.com/tourstream/kube-helper/blob/master/LICENSE
