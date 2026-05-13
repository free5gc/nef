#!/bin/bash
export PATH=$HOME/go/bin:$PATH
go install github.com/daixiang0/gci@latest
gci write internal/sbi/processor/ti_test.go
