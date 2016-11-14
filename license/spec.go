package license

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type licenseSpec struct {
	Key     string // SPDX ID of the license
	SHA1    string
	SHA256  string
	Aliases []string
}

var (
	licensesMap = licenseMap() // map from SPDX ID to spec for known licenses
	aliasesMap  = aliasMap()   // map from alias to SPDX ID for known licenses. Also contains entries that map SPDX IDs to themselves.
	// specs is a slice of licenseSpecs for known licenses that defines aliases and checksums.
	specs = []licenseSpec{
		{
			Key:     "agpl-3.0",
			SHA1:    "5835213bd72873e87ee97e546d023ca970bf8c08",
			SHA256:  "76a97c878c9c7a8321bb395c2b44d3fe2f8d81314d219b20138ed0e2dddd5182",
			Aliases: []string{"agpl"},
		},
		{
			Key:     "apache-2.0",
			SHA1:    "92170cdc034b2ff819323ff670d3b7266c8bffcd",
			SHA256:  "b40930bbcf80744c86c46a12bc9da056641d722716c378f5659b9e555ef833e1",
			Aliases: []string{"apache"},
		},
		{
			Key:     "bsd-2-clause",
			SHA1:    "a7e043c62ed66866b3f32f7d9561bc41a82ac970",
			SHA256:  "bc6da8e95c49652738b398592f5a89aaf1f168b478184d40b8177fdb49593ff5",
			Aliases: []string{"bsd-2"},
		},
		{
			Key:     "bsd-3-clause",
			SHA1:    "dddbe3da9055c57371f54f5a23143b5f1ea9f1f7",
			SHA256:  "c6bce241128aaf54728d86e9034e410385fda959073c467f377c4f4fa4253f69",
			Aliases: []string{"bsd-3"},
		},
		{
			Key:     "epl-1.0",
			SHA1:    "dddbe3da9055c57371f54f5a23143b5f1ea9f1f7",
			SHA256:  "c6bce241128aaf54728d86e9034e410385fda959073c467f377c4f4fa4253f69",
			Aliases: []string{"epl"},
		},
		{
			Key:    "gpl-2.0",
			SHA1:   "3127907a7623734f830e8c69ccee03b693bf993e",
			SHA256: "db296f2f7f35bca3a174efb0eb392b3b17bd94b341851429a3dff411b1c2fc73",
		},
		{
			Key:     "gpl-3.0",
			SHA1:    "12d81f50767d4e09aa7877da077ad9d1b915d75b",
			SHA256:  "589ed823e9a84c56feb95ac58e7cf384626b9cbf4fda2a907bc36e103de1bad2",
			Aliases: []string{"gpl"},
		},
		{
			Key:    "lgpl-2.1",
			SHA1:   "731a8eff333b8f7053ab2220511b524c87a75923",
			SHA256: "9b872a8a070b8ad329c4bd380fb1bf0000f564c75023ec8e1e6803f15364b9e9",
		},
		{
			Key:     "lgpl-3.0",
			SHA1:    "f45ee1c765646813b442ca58de72e20a64a7ddba",
			SHA256:  "da7eabb7bafdf7d3ae5e9f223aa5bdc1eece45ac569dc21b3b037520b4464768",
			Aliases: []string{"lgpl"},
		},
		{
			Key:    "mit",
			SHA1:   "2c87153926f8a458cffc9a435e15571ba721c2fa",
			SHA256: "002c2696d92b5c8cf956c11072baa58eaf9f6ade995c031ea635c6a1ee342ad1",
		},
		{
			Key:     "mpl-2.0",
			SHA1:    "d22157abc0fc0b4ae96380c09528e23cf77290a9",
			SHA256:  "1f256ecad192880510e84ad60474eab7589218784b9a50bc7ceee34c2b91f1d5",
			Aliases: []string{"mpl"},
		},
		{
			Key:    "unlicense",
			SHA1:   "24944bf7920108f5a4790e6071c32e9102760c37",
			SHA256: "88d9b4eb60579c191ec391ca04c16130572d7eedc4a86daa58bf28c6e14c9bcd",
		},
	}
)

// Aliases returns the aliases for the license with the provided SPDX ID.
func Aliases(licenseID string) []string {
	if spec, ok := licensesMap[strings.ToLower(licenseID)]; ok {
		return spec.Aliases
	}
	return nil
}

// Create returns the content of the requested license as a string. If the license is a templatized one that uses author
// information, the provided authorInfo is used to get the author information. If the license is not a templatized one,
// authorInfo can be nil. If the template uses author information and authorInfo is nil, the function returns an error.
// The provided key is the SPDX ID of the license or an alias for the license.
func Create(licenseKey string, cache Cache, authorInfo AuthorInfo) (string, error) {
	licenseKey = strings.ToLower(licenseKey)
	if v, ok := aliasesMap[licenseKey]; ok {
		// if provided key is an alias, look up the SPDX ID and use it
		licenseKey = v
	}

	license, err := cache.Get(licenseKey)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get content of license %s", licenseKey)
	}
	if hasAuthorInfo(license) {
		if authorInfo == nil {
			return "", errors.Errorf("%s license is templated with author information, but none was provided", licenseKey)
		}
		license = render(license, authorInfo)
	}
	return license, nil
}

func licenseMap() map[string]licenseSpec {
	m := make(map[string]licenseSpec)
	for _, l := range specs {
		m[l.Key] = l
	}
	return m
}

func aliasMap() map[string]string {
	m := make(map[string]string)
	for _, l := range specs {
		addMustUnique(m, l.Key, l.Key)
		for _, alias := range l.Aliases {
			addMustUnique(m, alias, l.Key)
		}
	}
	return m
}

func addMustUnique(m map[string]string, k, v string) {
	if lower := strings.ToLower(k); k != lower {
		panic(fmt.Sprintf("key must be lowercase, but %s != %s", k, lower))
	}
	if lower := strings.ToLower(v); v != lower {
		panic(fmt.Sprintf("value must be lowercase, but %s != %s", v, lower))
	}
	if vv, ok := m[k]; ok {
		// panic if key already exists in map
		panic(fmt.Sprintf("failed to add {%s: %s} to map because entry already exists: {%s: %s}", k, v, k, vv))
	}
	m[k] = v
}

type AuthorInfo interface {
	FullName() string
	Year() string
}

type authorInfoStruct struct {
	fullName string
	year     string
}

func (a *authorInfoStruct) FullName() string {
	return a.fullName
}

func (a *authorInfoStruct) Year() string {
	return a.year
}

// NewAuthorInfo returns a new AuthorInfo that represents the information for the provided author name, created year and
// updated year. Returns nil if the authorName is the empty string. If the created year and updated year are the same
// values, the single value is used, otherwise, the year is represented as the range {{createdYear}}-{{updatedYear}}.
func NewAuthorInfo(authorName string, createdYear, updatedYear int) AuthorInfo {
	if authorName == "" {
		return nil
	}
	year := fmt.Sprintf("%d", createdYear)
	// if last updated year is different from created year, add range
	if updatedYear != createdYear {
		year += fmt.Sprintf("-%d", updatedYear)
	}
	return &authorInfoStruct{
		year:     year,
		fullName: authorName,
	}
}

func hasAuthorInfo(licenseContent string) bool {
	return strings.Contains(licenseContent, "[fullname]") || strings.Contains(licenseContent, "[year]")
}

func render(content string, author AuthorInfo) string {
	return strings.NewReplacer(
		"[fullname]", author.FullName(),
		"[year]", author.Year(),
	).Replace(content)
}
