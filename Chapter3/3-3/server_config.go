package main

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func validateServerConfig(addr, tlsCert, tlsKey, googleRedirectURL string) error {
	_, serverPort, err := net.SplitHostPort(addr)
	if err != nil {
		return errors.New("server address must include a host and port")
	}
	portNumber, err := strconv.Atoi(serverPort)
	if err != nil || portNumber < 0 || portNumber > 65535 {
		return errors.New("server port must be numeric and between 0 and 65535")
	}
	if (tlsCert == "") != (tlsKey == "") {
		return errors.New("TLS certificate and key must be configured together")
	}
	if tlsCert == "" && !isLoopbackAddress(addr) {
		return errors.New("plain HTTP is allowed only on a loopback address")
	}

	redirect, err := url.Parse(googleRedirectURL)
	if err != nil || redirect.Host == "" || redirect.User != nil || redirect.RawQuery != "" || redirect.Fragment != "" || redirect.Path != "/auth/callback/google" {
		return errors.New("Google redirect URL must be an absolute callback URL without userinfo, query, or fragment")
	}
	if tlsCert != "" && redirect.Scheme != "https" {
		return errors.New("HTTPS deployments require an HTTPS Google redirect URL")
	}
	switch redirect.Scheme {
	case "https":
	case "http":
		if !isLoopbackAddress(addr) || !isLoopbackHost(redirect.Hostname()) {
			return errors.New("HTTP OAuth callbacks are allowed only for loopback development")
		}
		if redirect.Port() != serverPort {
			return errors.New("loopback HTTP OAuth callback port must match the server port")
		}
	default:
		return errors.New("Google redirect URL must use HTTP or HTTPS")
	}
	return nil
}

func isLoopbackAddress(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	return err == nil && isLoopbackHost(host)
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
