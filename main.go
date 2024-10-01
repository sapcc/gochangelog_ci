/******************************************************************************
*
*  Copyright 2023 SAP SE
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
******************************************************************************/

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/shurcooL/githubv4"
	"golang.org/x/mod/modfile"
	"golang.org/x/oauth2"
)

var githubToken = flag.String("github-token", os.Getenv("GITHUB_TOKEN"), "A personal GitHub access token (required)")
var goModFile = flag.String("go-mod-file", "go.mod", "path to go.mod file (default: go.mod)")
var branch = flag.String("branch", "master", "branch to generate changelog for (default: master)")
var repoOwner = flag.String("repo-owner", "", "owner of the repo")
var repoName = flag.String("repo-name", "", "name of the repo")
var repoURL = flag.String("repo-url", "", "url of the repo, overrides repo-owner and repo-name")
var flagNoColor = flag.Bool("no-color", false, "Disable color output")
var version = flag.Bool("version", false, "Print version and exit")

type gitHistory []struct {
	OID     string
	Message string
}

type ghClient struct {
	*githubv4.Client
}

func (gh *ghClient) commit2time(commit string) (time.Time, error) {
	var q struct {
		Repository struct {
			Object *struct {
				Commit struct {
					CommittedDate time.Time
				} `graphql:"... on Commit"`
			} `graphql:"object(expression: $commit)"`
		} `graphql:"repository(owner: $repoOwner, name: $repoName)"`
	}

	if err := gh.Query(context.Background(), &q, map[string]interface{}{
		"repoOwner": githubv4.String(*repoOwner),
		"repoName":  githubv4.String(*repoName),
		"commit":    githubv4.String(commit),
	}); err != nil {
		return time.Time{}, err
	}
	if q.Repository.Object == nil {
		return time.Time{}, errors.New("unable to find commit")
	}
	return q.Repository.Object.Commit.CommittedDate, nil
}

func (gh *ghClient) getDiff(since, until time.Time) (gitHistory, error) {
	var q struct {
		Repository struct {
			Object struct {
				Commit struct {
					History struct {
						Nodes gitHistory
					} `graphql:"history(since: $since, until: $until)"`
				} `graphql:"... on Commit"`
			} `graphql:"object(expression: $branch)"`
		} `graphql:"repository(owner: $repoOwner, name: $repoName)"`
	}
	if err := gh.Query(context.Background(), &q, map[string]interface{}{
		"repoOwner": githubv4.String(*repoOwner),
		"repoName":  githubv4.String(*repoName),
		"branch":    githubv4.String(*branch),
		"since":     githubv4.GitTimestamp{Time: since},
		"until":     githubv4.GitTimestamp{Time: until},
	}); err != nil {
		return nil, err
	}

	return q.Repository.Object.Commit.History.Nodes, nil
}

func (gh *ghClient) getFile(ref string) ([]byte, error) {
	var q struct {
		Repository struct {
			Object struct {
				Blob struct {
					Text string
				} `graphql:"... on Blob"`
			} `graphql:"object(expression: $expression)"`
		} `graphql:"repository(owner: $repoOwner, name: $repoName)"`
	}
	if err := gh.Query(context.Background(), &q, map[string]interface{}{
		"repoOwner":  githubv4.String(*repoOwner),
		"repoName":   githubv4.String(*repoName),
		"expression": githubv4.String(ref),
	}); err != nil {
		return nil, err
	}

	return []byte(q.Repository.Object.Blob.Text), nil
}

func main() {
	flag.Parse()

	// for **** go-makefile-maker dockerfile to work
	if *version {
		print("Version: SomeVersion(tm)\n")
		os.Exit(0)
	}

	if flag.NArg() != 2 {
		flag.Usage()
		print("  required positional argument: Commit-A Commit-B\n")
		os.Exit(1)
	}
	if *flagNoColor {
		color.NoColor = true // disables colorized output
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *githubToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	gh := ghClient{githubv4.NewClient(httpClient)}

	if repoURL != nil && *repoURL != "" {
		u, err := url.Parse(*repoURL)
		if err != nil {
			panic(err)
		}
		p := strings.Split(strings.Trim(u.Path, "/"), "/")
		repoOwner = &p[0]
		repoName = &p[1]

		if u.Host != "github.com" {
			gh = ghClient{githubv4.NewEnterpriseClient(u.Scheme+"://"+u.Host+"/api/graphql", httpClient)}
		}
	}

	commitA := flag.Arg(0)
	commitB := flag.Arg(1)

	since, err := gh.commit2time(commitA)
	if err != nil {
		log.Fatal(err)
	}
	until, err := gh.commit2time(commitB)
	if err != nil {
		log.Fatal(err)
	}

	history, err := gh.getDiff(since, until)
	if err != nil {
		log.Fatal(err)
	}

	// Print Changelog
	fmt.Println("Changelog:")
	for _, n := range history {
		if n.OID == commitA {
			// skip "from commit"
			continue
		}
		fmt.Printf("  %s %s\n", n.OID, color.YellowString(n.Message))
	}

	// Print go.mod changes
	nameA := fmt.Sprintf("%s:%s", commitA, *goModFile)
	contentA, err := gh.getFile(nameA)
	if err != nil {
		log.Fatal(err)
	}
	goModA, err := modfile.Parse(nameA, contentA, nil)
	if err != nil {
		log.Fatal(err)
	}

	nameB := fmt.Sprintf("%s:%s", commitB, *goModFile)
	contentB, err := gh.getFile(nameB)
	if err != nil {
		log.Fatal(err)
	}
	goModB, err := modfile.Parse(nameB, contentB, nil)
	if err != nil {
		log.Fatal(err)
	}

	if goModA.Go == nil || goModB.Go == nil {
		return
	}

	fmt.Println("Changes in go.mod:")
	if goModA.Go.Version != goModB.Go.Version {
		color.Yellow("  go version changed from %s to %s\n", goModA.Go.Version, goModB.Go.Version)
	}

	// changed and removed modules
	for _, req := range goModA.Require {
		modPathFound := false
		for _, req2 := range goModB.Require {
			if req.Mod.Path == req2.Mod.Path {
				modPathFound = true
				if req.Mod.Version != req2.Mod.Version {
					fmt.Printf("  %s version changed from %s to %s\n",
						color.YellowString(req.Mod.Path), req.Mod.Version, req2.Mod.Version)
				}
			}
		}
		if !modPathFound {
			fmt.Printf("  %s removed\n", color.RedString(req.Mod.Path))
		}
	}

	// added modules
	for _, req := range goModB.Require {
		modPathFound := false
		for _, req2 := range goModA.Require {
			if req.Mod.Path == req2.Mod.Path {
				modPathFound = true
			}
		}
		if !modPathFound {
			fmt.Printf("  %s added\n", color.GreenString(req.Mod.Path))
		}
	}
}
