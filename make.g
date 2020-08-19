#!/usr/local/bin/gentee

// This script builds and runs Eonza application
// It uses Gentee programming language - https://github.com/gentee/gentee

run {
    str env = $ go env
    $GOPATH = RegExp(env, `GOPATH="?([^"|\n|\r]*)`)

    $ /home/ak/go/bin/esc -ignore "\.git|LICENSE|README.md" -o assets.go ../eonza-assets 
//    $ go install -ldflags "-s -w" -tags "eonza standard"
    $ go install -ldflags "-s -w"
    $ cp ${GOPATH}/bin/eonza /home/ak/app/eonza-dev/eonza
    $ /home/ak/app/eonza-dev/eonza
//    $ /home/ak/app/eonza/eonza -cfg /home/ak/app/eonza/config
//  /home/ak/go/bin/esc -o assets.go ../eonza-assets/default
}