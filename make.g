#!/usr/local/bin/gentee

// This script builds and runs Eonza application
// It uses Gentee programming language - https://github.com/gentee/gentee

run {
    str env = $ go env
    $GOPATH = RegExp(env, `GOPATH="?([^"|\n|\r]*)`)

    $ go install 
    $ cp ${GOPATH}/bin/eonza /home/ak/app/eonza/eonza
    $ /home/ak/app/eonza/eonza
    $ /home/ak/app/eonza/eonza -cfg /home/ak/app/eonza/config
}