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

## Usage

1. install gcloud sdk
2. run `gcloud auth application-default login` to create default credentials

[coveralls-image]: https://coveralls.io/repos/github/tourstream/kube-helper/badge.svg
[coveralls-url]: https://coveralls.io/github/tourstream/kube-helper

[travis-image]: https://travis-ci.org/tourstream/kube-helper.svg?branch=master
[travis-url]: https://travis-ci.org/tourstream/kube-helper

[license-image]: https://img.shields.io/github/license/tourstream/kube-helper.svg?style=flat-square
[license-url]: https://github.com/tourstream/kube-helper/blob/master/LICENSE

## Tests

To generate mocks use the following tool

    https://github.com/vektra/mockery


Example command to generate mocks for e.g. command folder

    mockery -dir service -name ApplicationServiceInterface -output _mocks -outpkg _mocks
