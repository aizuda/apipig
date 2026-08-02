package service

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

// validateRepositoryURL rejects URL forms that could expose credentials or
// turn the review worker into a local-file or internal-network client.
func validateRepositoryURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" {
		return nil, errors.New("仓库地址必须是有效的 HTTPS URL")
	}
	if parsed.User != nil {
		return nil, errors.New("仓库地址不能内嵌用户名或密码，请使用仓库 Token")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("仓库地址不能包含查询参数或片段")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return nil, errors.New("仓库地址不能指向本机或内部网络")
	}
	if address, parseErr := netip.ParseAddr(host); parseErr == nil && !isPublicRepositoryAddress(address.Unmap()) {
		return nil, errors.New("仓库地址不能指向本机或内部网络")
	}
	return parsed, nil
}

// validateRepositoryRemote resolves the hostname immediately before Git uses
// it. This catches private DNS targets even when the project was saved earlier.
func validateRepositoryRemote(ctx context.Context, rawURL string) error {
	parsed, err := validateRepositoryURL(rawURL)
	if err != nil {
		return err
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", parsed.Hostname())
	if err != nil || len(addresses) == 0 {
		return errors.New("无法解析仓库地址")
	}
	for _, address := range addresses {
		if !isPublicRepositoryAddress(address.Unmap()) {
			return errors.New("仓库地址解析到了本机或内部网络")
		}
	}
	return nil
}

func isPublicRepositoryAddress(address netip.Addr) bool {
	if !address.IsValid() {
		return false
	}
	if address.IsUnspecified() || address.IsLoopback() || address.IsPrivate() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() {
		return false
	}
	for _, prefix := range nonPublicRepositoryPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

var nonPublicRepositoryPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}
