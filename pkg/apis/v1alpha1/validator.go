package v1alpha1

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/tacokumo/portal-api/pkg/apis/v1alpha1/api"
)

var (
	applicationNameRegexp = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	secretKeyRegexp       = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

	privateRanges []*net.IPNet
)

func init() {
	cidrs := []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"0.0.0.0/8",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range cidrs {
		_, ipNet, _ := net.ParseCIDR(cidr)
		privateRanges = append(privateRanges, ipNet)
	}
}

func isPrivateIP(ip net.IP) bool {
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

func validationError(msg string) *ErrorWithCode {
	return &ErrorWithCode{Code: http.StatusBadRequest, Message: msg}
}

func ValidateApplicationName(name string) error {
	if !applicationNameRegexp.MatchString(name) {
		return validationError("application name must be 1-63 characters, lowercase alphanumeric or '-', and must start and end with an alphanumeric character")
	}
	return nil
}

func ValidateRepositoryURL(rawURL string) error {
	if rawURL == "" {
		return validationError("repository_url is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return validationError("repository_url is not a valid URL")
	}

	if parsed.Scheme != "https" {
		return validationError("repository_url must use https scheme")
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return validationError("repository_url must have a hostname")
	}

	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") {
		return validationError("repository_url must not point to localhost")
	}

	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateIP(ip) {
			return validationError("repository_url must not point to a private or reserved IP address")
		}
		return nil
	}

	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return nil
	}
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && isPrivateIP(ip) {
			return validationError("repository_url must not resolve to a private or reserved IP address")
		}
	}

	return nil
}

func ValidateAppconfigPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return validationError("appconfig_path is required")
	}
	if len(path) > 512 {
		return validationError("appconfig_path must be at most 512 characters")
	}
	if strings.Contains(path, "..") {
		return validationError("appconfig_path must not contain '..' (path traversal)")
	}
	if strings.ContainsRune(path, 0) {
		return validationError("appconfig_path must not contain null bytes")
	}
	return nil
}

func ValidateAppconfigBranch(branch string) error {
	if strings.TrimSpace(branch) == "" {
		return validationError("appconfig_branch is required")
	}
	if len(branch) > 253 {
		return validationError("appconfig_branch must be at most 253 characters")
	}

	for _, ch := range branch {
		if ch <= 0x1f || ch == 0x7f {
			return validationError("appconfig_branch must not contain control characters")
		}
	}

	for _, forbidden := range []string{"..", "~", "^", ":", " ", "\\", "?", "*", "[", "@{"} {
		if strings.Contains(branch, forbidden) {
			return validationError(fmt.Sprintf("appconfig_branch must not contain '%s'", forbidden))
		}
	}

	if strings.HasPrefix(branch, ".") || strings.HasPrefix(branch, "/") {
		return validationError("appconfig_branch must not start with '.' or '/'")
	}
	if strings.HasSuffix(branch, ".") || strings.HasSuffix(branch, "/") {
		return validationError("appconfig_branch must not end with '.' or '/'")
	}
	if strings.HasSuffix(branch, ".lock") {
		return validationError("appconfig_branch must not end with '.lock'")
	}

	return nil
}

func ValidateSecretKey(key string) error {
	if key == "" {
		return validationError("secret key is required")
	}
	if len(key) > 253 {
		return validationError("secret key must be at most 253 characters")
	}
	if !secretKeyRegexp.MatchString(key) {
		return validationError("secret key must contain only alphanumeric characters, '-', '_', or '.'")
	}
	return nil
}

func ValidateSecretValue(value string) error {
	if value == "" {
		return validationError("secret value must not be empty")
	}
	if len(value) > 1048576 {
		return validationError("secret value must be at most 1MB")
	}
	return nil
}

func ValidateCreateApplicationRequest(req *api.CreateApplicationRequest) error {
	if err := ValidateApplicationName(req.Name); err != nil {
		return err
	}
	if err := ValidateRepositoryURL(req.RepositoryURL); err != nil {
		return err
	}
	if err := ValidateAppconfigPath(req.AppconfigPath); err != nil {
		return err
	}
	if err := ValidateAppconfigBranch(req.AppconfigBranch); err != nil {
		return err
	}
	return nil
}

func ValidateUpdateApplicationRequest(req *api.UpdateApplicationRequest) error {
	if err := ValidateRepositoryURL(req.RepositoryURL); err != nil {
		return err
	}
	if err := ValidateAppconfigPath(req.AppconfigPath); err != nil {
		return err
	}
	if err := ValidateAppconfigBranch(req.AppconfigBranch); err != nil {
		return err
	}
	return nil
}

func ValidateCreateSecretRequest(req *api.CreateSecretRequest) error {
	if len(req.Items) == 0 {
		return validationError("at least one secret item is required")
	}
	for i, item := range req.Items {
		if err := ValidateSecretKey(item.Key); err != nil {
			return validationError(fmt.Sprintf("items[%d]: %s", i, err.(*ErrorWithCode).Message))
		}
		if err := ValidateSecretValue(item.Value); err != nil {
			return validationError(fmt.Sprintf("items[%d]: %s", i, err.(*ErrorWithCode).Message))
		}
	}
	return nil
}
