#!/bin/bash

gofmt -l -s -w .
golint .
