<!--
SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company

SPDX-License-Identifier: Apache-2.0
-->

# gochangelog_ci

A changelog generator for GitHub based Go projects that uses the GitHub API to generate a changelog based on the commit messages.
It is intended to be used in a CI environment to display a changelog before a release/deployment.

## Usage

```bash
Usage of gochangelog_ci:
  -branch string
    	branch to generate changelog for (default: master) (default "master")
  -github-token string
    	A personal GitHub access token (required)
  -go-mod-file string
    	path to go.mod file (default: go.mod) (default "go.mod")
  -no-color
    	Disable color output
  -repo-name string
    	name of the repo
  -repo-owner string
    	owner of the repo
  required positional argument: Commit-A Commit-B
```

## Example

```bash
$ gochangelog_ci -token $GITHUB_TOKEN -repo-name=archer -repo-owner=sapcc -branch=main b2a22ec351fecbe9bbb380b08b36864659e19905 65acbf11a22660d93944a946c2e539f6c1a89dcf
Changelog:
  65acbf11a22660d93944a946c2e539f6c1a89dcf fix(deps): update module github.com/go-co-op/gocron to v1.34.2
  9e37f91a56b8afaef0bfeaa4e021984f9aad8c04 fix(deps): update module github.com/go-co-op/gocron to v1.34.1
  a8ca254f984eab5ae293c230d475f5aa1ebf5a2d fix(deps): update github.com/sapcc/go-bits digest to bf2c075
  4583b9975c26eed1d9cbf5136c4eaec08f0bfe3b [endpoints] fix port ip discovery for endpoints
  bd125395bf98ef2757a1e736aa8920d9593a4d2b [as3] fix member enable=true
  b2a22ec351fecbe9bbb380b08b36864659e19905 [release action] remove goversion completely
Changes in go.mod:
  github.com/go-co-op/gocron version changed from v1.34.0 to v1.34.2
  github.com/sapcc/go-bits version changed from v0.0.0-20230912142452-3a5bef45cdb0 to v0.0.0-20230920131001-bf2c07577372
```

## License
Apache License 2.0
