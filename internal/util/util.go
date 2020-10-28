package util

import (
	"bitbucket.org/free5gc-team/path_util"
)

// Path of HTTP2 key and log file
var (
	NEF_LOG_PATH    = path_util.Free5gcPath("free5gc/nefsslkey.log")
	NEF_PEM_PATH    = path_util.Free5gcPath("free5gc/support/TLS/nef.pem")
	NEF_KEY_PATH    = path_util.Free5gcPath("free5gc/support/TLS/nef.key")
	NEF_CONFIG_PATH = path_util.Free5gcPath("free5gc/config/nefcfg.conf")
	NEF_BASIC_PATH  = "https://localhost:29505"
)
