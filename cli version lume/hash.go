package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyAndHashFile(ctx context.Context, src, dst string, mode os.FileMode) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return "", err
	}

	var success bool
	defer func() {
		out.Close()
		if !success {
			os.Remove(dst)
		}
	}()

	h := sha256.New()
	buf := make([]byte, 64*1024)

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		n, readErr := in.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return "", writeErr
			}
			h.Write(buf[:n])
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}

	if err := out.Sync(); err != nil {
		return "", fmt.Errorf("dosya senkronize edilemedi: %w", err)
	}

	if err := out.Close(); err != nil {
		return "", fmt.Errorf("dosya kapatılamadı: %w", err)
	}
	success = true

	if err := os.Chmod(dst, mode); err != nil {
		fmt.Printf("[WARN]  %s: İzinler ayarlanamadı: %v\n", filepath.Base(dst), err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
