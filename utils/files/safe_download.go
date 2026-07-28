package files

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var blockedDownloadPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}

func DownloadFileWithProcessSafe(rawURL, dst, key string, ignoreCertificate bool) error {
	client := newSafeDownloadClient(ignoreCertificate)
	request, err := newSafeDownloadRequest(context.Background(), rawURL)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	if err := validateDownloadResponse(resp); err != nil {
		resp.Body.Close()
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		resp.Body.Close()
		return err
	}
	go func() {
		defer out.Close()
		defer resp.Body.Close()
		counter := &WriteCounter{Key: key, Name: filepath.Base(dst)}
		if resp.ContentLength > 0 {
			counter.Total = uint64(resp.ContentLength)
		}
		if _, err := io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
			return
		}
		counter.Total = counter.Written
		counter.SaveProcess()
	}()
	return nil
}

func DownloadFileWithCallbackSafe(ctx context.Context, rawURL, dst string, ignoreCertificate bool, progressFn func(written, total uint64)) error {
	client := newSafeDownloadClient(ignoreCertificate)
	request, err := newSafeDownloadRequest(ctx, rawURL)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := validateDownloadResponse(resp); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	var total uint64
	if resp.ContentLength > 0 {
		total = uint64(resp.ContentLength)
	}
	reader := &progressCallbackReader{ctx: ctx, reader: resp.Body, total: total, progressFn: progressFn}
	if _, err := io.Copy(out, reader); err != nil {
		if errors.Is(err, context.Canceled) {
			_ = os.Remove(dst)
		}
		return err
	}
	return nil
}

func validateDownloadURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("download URL must use http or https")
	}
	if parsed.Hostname() == "" {
		return errors.New("download URL host is required")
	}
	if parsed.User != nil {
		return errors.New("download URL credentials are not allowed")
	}
	if address, err := netip.ParseAddr(parsed.Hostname()); err == nil && isBlockedDownloadAddr(address) {
		return fmt.Errorf("download URL address %q is blocked", address)
	}
	return nil
}

func isBlockedDownloadAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsValid() || !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return true
	}
	for _, prefix := range blockedDownloadPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func safeDownloadDialer() func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("resolve download host: %w", err)
		}
		for _, address := range addresses {
			if isBlockedDownloadAddr(address) {
				continue
			}
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(address.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			err = dialErr
		}
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("download host %q resolves only to blocked addresses", host)
	}
}

func newSafeDownloadClient(ignoreCertificate bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = safeDownloadDialer()
	transport.ResponseHeaderTimeout = 30 * time.Second
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: ignoreCertificate} // #nosec G402 -- explicit user option
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many download redirects")
			}
			return validateDownloadURL(req.URL.String())
		},
	}
}

func newSafeDownloadRequest(ctx context.Context, rawURL string) (*http.Request, error) {
	if err := validateDownloadURL(strings.TrimSpace(rawURL)); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept-Encoding", "identity")
	return request, nil
}

func validateDownloadResponse(resp *http.Response) error {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download request returned HTTP %d", resp.StatusCode)
	}
	return nil
}
