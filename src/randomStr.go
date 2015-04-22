package main

import (
	"math/rand"
	"strconv"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomStr() string { return strconv.FormatInt(r.Int63(), 36) }
