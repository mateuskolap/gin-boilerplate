package security

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"gin-boilerplate/internal/domain/port"
)

const (
	pwnedPasswordsEndpoint = "https://api.pwnedpasswords.com/range/"
	maxResponseBytes       = 256 * 1024
)

type PwnedPasswordChecker struct {
	client *http.Client
}

var _ port.CompromisedPasswordChecker = (*PwnedPasswordChecker)(nil)

func NewPwnedPasswordChecker() *PwnedPasswordChecker {
	return &PwnedPasswordChecker{client: &http.Client{Timeout: 2 * time.Second}}
}

func (c *PwnedPasswordChecker) IsCompromised(ctx context.Context, password string) (bool, error) {
	userAgent, err := applicationUserAgent()
	if err != nil {
		return false, err
	}
	hash := sha1.Sum([]byte(password))
	encodedHash := strings.ToUpper(hex.EncodeToString(hash[:]))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, pwnedPasswordsEndpoint+encodedHash[:5], nil)
	if err != nil {
		return false, err
	}
	request.Header.Set("Add-Padding", "true")
	request.Header.Set("User-Agent", userAgent)
	response, err := c.client.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, errors.New("HIBP returned a non-200 response")
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return false, err
	}
	if len(body) > maxResponseBytes {
		return false, errors.New("HIBP response exceeded the size limit")
	}

	wantedSuffix := encodedHash[5:]
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 1024), 128)
	for scanner.Scan() {
		suffix, countText, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			return false, errors.New("HIBP returned an invalid response")
		}
		count, err := strconv.Atoi(countText)
		if err != nil {
			return false, errors.New("HIBP returned an invalid response")
		}
		if count > 0 && strings.EqualFold(suffix, wantedSuffix) {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func applicationUserAgent() (string, error) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok || buildInfo.Main.Path == "" {
		return "", errors.New("application module name is unavailable")
	}
	return path.Base(buildInfo.Main.Path), nil
}
