//+build !windows,!nacl,!plan9

package zerologger

import (
	"io"
	"log/syslog"
	"regexp"
)

const defaultSyslogPriority = syslog.LOG_LOCAL0 | syslog.LOG_DEBUG

const DefaultSyslogNetwork = "udp"

var addrRegex = regexp.MustCompile(`^((ip|tcp|udp)(|4|6)|unix|unixgram|unixpacket):`)

func toNetworkAndAddress(s string) (string, string) {
	indexes := addrRegex.FindStringSubmatchIndex(s)
	if len(indexes) == 0 {
		return DefaultSyslogNetwork, s
	}
	return s[:indexes[3]], s[indexes[3]+1:]
}

func ConnectSyslogByParam(outputParam, tag string) (io.Writer, error) {
	if len(outputParam) == 0 || outputParam == "localhost" {
		return ConnectDefaultSyslog(tag)
	}

	nw, addr := toNetworkAndAddress(outputParam)
	return ConnectRemoteSyslog(nw, addr, tag)
}

func ConnectDefaultSyslog(tag string) (io.Writer, error) {
	w, err := syslog.New(defaultSyslogPriority, tag)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func ConnectRemoteSyslog(network, raddr string, tag string) (io.Writer, error) {
	w, err := syslog.Dial(network, raddr, defaultSyslogPriority, tag)
	if err != nil {
		return nil, err
	}
	return w, nil
}
