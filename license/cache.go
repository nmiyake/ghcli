// Copyright 2016 Nick Miyake. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root
// for license information.

package license

import (
	"crypto/sha1"
	"encoding/hex"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

type Cache interface {
	// Get returns the content of the license with the provided key (SPDX ID). Uses the GitHub API to get the
	// content of the license if it is not already cached.
	Get(licenseKey string) (string, error)
}

func NewCache(client *github.Client) Cache {
	return &cache{
		client: client,
		cache:  make(map[string]string),
	}
}

type cache struct {
	client *github.Client
	cache  map[string]string
}

func (c *cache) Get(licenseKey string) (string, error) {
	if l, ok := c.cache[licenseKey]; ok {
		return l, nil
	}
	l, _, err := c.client.Licenses.Get(licenseKey)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get license %s", licenseKey)
	}
	if spec, ok := licensesMap[licenseKey]; ok {
		// if spec exists for license, verify hash of retrieved license against spec
		sha1bytes := sha1.Sum([]byte(*l.Body))
		sha1sum := hex.EncodeToString(sha1bytes[:])
		if spec.SHA1 != sha1sum {
			return "", errors.Wrapf(err, "SHA-1 sums for license %s does not match: expected %s, was %s", spec.Key, spec.SHA1, sha1sum)
		}
	}
	c.cache[licenseKey] = *l.Body
	return c.cache[licenseKey], nil
}
