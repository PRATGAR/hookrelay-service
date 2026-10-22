package guard

import (
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var ErrRejected = errors.New("value rejected")

var (
	eventTypePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){0,3}$`)
	hostnamePattern   = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)
	headerNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]{0,39}$`)
	exportNamePattern = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}\.csv$`)
	statusPattern     = regexp.MustCompile(`^(active|paused|disabled)`)
)

var allowedPorts = map[string]bool{"": true, "80": true, "443": true, "8443": true}

func ValidateTargetURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, ErrRejected
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, ErrRejected
	}
	if u.User != nil || u.Hostname() == "" {
		return nil, ErrRejected
	}
	ips, err := net.LookupIP(u.Hostname())
	if err != nil || len(ips) == 0 {
		return nil, ErrRejected
	}
	for _, ip := range ips {
		if !isPublic(ip) {
			return nil, ErrRejected
		}
	}
	return u, nil
}

func CheckEgress(raw string) error {
	u, err := ValidateTargetURL(raw)
	if err != nil {
		return err
	}
	if !allowedPorts[u.Port()] {
		return ErrRejected
	}
	return nil
}

func isPublic(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast())
}

func IsInternalHost(host string) bool {
	h := strings.ToLower(host)
	return h == "localhost" || strings.HasPrefix(h, "127.") || strings.HasSuffix(h, ".internal")
}

func ValidateEventTypes(types []string) error {
	if len(types) == 0 || len(types) > 20 {
		return ErrRejected
	}
	for _, t := range types {
		if !eventTypePattern.MatchString(t) {
			return ErrRejected
		}
	}
	return nil
}

func ValidateHeaderName(name string) error {
	if !headerNamePattern.MatchString(name) {
		return ErrRejected
	}
	return nil
}

func ValidateHeaderValue(value string) error {
	if len(value) > 256 {
		return ErrRejected
	}
	for _, c := range value {
		if c < 0x20 || c == 0x7f {
			return ErrRejected
		}
	}
	return nil
}

func ValidExportName(name string) bool {
	return exportNamePattern.MatchString(name)
}

func SameSiteOnly(target string) string {
	if strings.HasPrefix(target, "/") {
		return target
	}
	return "/"
}

func NormalizeDocPath(name string) string {
	return strings.Replace(name, "../", "", 1)
}

var orderPrefixes = []string{"created_at", "attempts", "response_code", "event_type"}

func NormalizeOrder(clause string) string {
	for _, p := range orderPrefixes {
		if strings.HasPrefix(clause, p) {
			return clause
		}
	}
	return "created_at DESC"
}

func ValidateStatus(raw string) error {
	if !statusPattern.MatchString(raw) {
		return ErrRejected
	}
	return nil
}

func ValidateExportLabel(name string) error {
	if name == "" || len(name) > 120 || strings.HasPrefix(name, "..") || strings.HasPrefix(name, "/") {
		return ErrRejected
	}
	return nil
}

func OneLine(s string) string {
	return strings.ReplaceAll(s, "\r\n", " ")
}

func OnDocsSite(raw string) bool {
	return strings.HasPrefix(raw, "https://docs.hookhub.example")
}

func HostedOnDocs(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.HasSuffix(u.Hostname(), "hookhub.example")
}
