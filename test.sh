#!/usr/bin/env bash

# this script is specified as the 'tests' task in .circleci/config.yml

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )" # get dir containing this script
cd $DIR # always from from script dir

# set error on return if any part of a pipe command fails
set -o pipefail

# base go test command
GO_TEST="go test -v -timeout=10m -vet=off"

# list modules for CI testing
function listModules() {
  glide nv | grep -v Utilities | grep -v longTest | grep -v peerTest | grep -v simTest | grep -v elections | grep -v activations | grep -v netTest | grep "\.\.\."
}

# load a list of tests to execute
function loadTestList() {
  case $1 in
    all ) # run all unit tests at once
      TESTS=("./...")
      ;;
    unittest ) # run unit test in batches
      TESTS=$({ \
        listModules ; \
      })
      ;;
    peertest ) # run only peer tests
      TESTS=$({ \
        ls peerTest/*A_test.go; \
      })
      ;;

    simtest ) # run only simulation tests
      TESTS=$({ \
        go test --tags=simtest --list=Test ./engine/... | awk '/^Test/ { print "engine/"$1 }' ; \
        go test --tags=simtest --list=Test ./simTest/... | awk '/^Test/ { print "simTest/"$1 }' ; \
      })
      ;;

    "" ) # run everything

      if [[ "${CI}x" ==  "x" ]] ; then
        # running locally
        TESTS=$({ \
          listModules ; \
          go test --tags=simtest --list=Test ./engine/... | awk '/^Test/ { print "engine/"$1 }' ; \
          go test --tags=simtest --list=Test ./simTest/... | awk '/^Test/ { print "simTest/"$1 }' ; \
          ls peerTest/*A_test.go; \
        })
      else
        # running on circle
        TESTS=$({ \
          listModules ; \
          go test --tags=simtest --list=Test ./engine/... | awk '/^Test/ { print "engine/"$1 }' ; \
          go test --tags=simtest --list=Test ./simTest/... | awk '/^Test/ { print "simTest/"$1 }' ; \
          ls peerTest/*A_test.go; \
        } | circleci tests split ) # circleci helper spreads tests across containers
      fi
      ;;

    * )
      echo "Unknown option" $1
      exit -1
      ;;
  esac
}

function runTests() {

  loadTestList $1

  FAILURES=()
  FAIL=""

  echo '---------------'
  echo "${TESTS}"
  echo '---------------'

  for TST in ${TESTS[*]} ; do
    case `dirname $TST` in
      engine )
        testEngine $TST
        ;;
      simTest )
        testSim $TST
        ;;
      peerTest )
        testPeer $TST
        ;;
      * ) # package name provided instead
        unitTest $TST
        ;;
    esac

    if [[ $? != 0 ]] ;  then
      FAIL=1
      FAILURES+=($TST)
    fi
  done

  if [[ "${FAIL}x" != "x" ]] ; then
    echo "TESTS FAIL"
    echo '---------------'
    for F in ${FAILURES[*]} ; do
      echo $F
    done
    exit 1
  else
    echo "ALL TESTS PASS"
    exit 0
  fi
}

# run A/B peer coodinated tests
# $1 should be a path to a test file
function testPeer() {
  A=${1/B_/A_}
  B=${1/A_/B_}

  # run part A in background
  nohup $GO_TEST $A &> a_testout.txt &

  # run part B in foreground
  $GO_TEST $B &> b_testout.txt
}

# run unit tests per module
function unitTest() {
  $GO_TEST -tags=all $1 | egrep 'PASS|FAIL|RUN'
}

# run a simtest 
# $1 matches simTest/<TestSomeTestName>
function testSim() {
  $GO_TEST -tags=simtest -run=${1/simTest\//} ./simTest/... | egrep 'PASS|FAIL|RUN'
}

# run a simtest from engine package
# $1 matches engine/<TestSomeTestName>
function testEngine() {
  $GO_TEST -tags=simtest -run=${1/engine\//} ./engine/... | egrep 'PASS|FAIL|RUN'
}

runTests $1
