// Copyright 2016 Mathieu Lonjaret

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	flagHelp  = flag.Bool("h", false, "show this help")
	flagUser  = flag.String("user", "", "github username")
	flagToken = flag.String("token", "", "OAuth token. Do the full OAuth dance to get one, or generate a personal API token at https://github.com/settings/tokens")
	flagForks = flag.Bool("forks", false, "Fetches forked repos as well.")
	// TODO(mpl): make it a list of comma separated items.
	flagRepo    = flag.String("repo", "", "Additional repo (name) to fetch, even though flagForks says not to.")
	flagVerbose = flag.Bool("v", false, "verbose")
)

func usage() {
	fmt.Fprintf(os.Stderr, "\t igotyourback -user mpl -token oauthTokenHere\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *flagHelp {
		usage()
	}
	if *flagUser == "" {
		usage()
	}
	if *flagToken == "" {
		usage()
	}
	nargs := flag.NArg()
	if nargs > 0 {
		usage()
	}

	// or see https://github.com/google/go-github/blob/master/examples/basicauth/main.go
	// for a basic auth example instead.
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *flagToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	cl := github.NewClient(tc)

	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 20},
	}
	var allRepos []*github.Repository
	for {
		repos, resp, err := cl.Repositories.List(*flagUser, opt)
		if err != nil {
			log.Fatal(err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	for _, repo := range allRepos {
		// Because it's simpler to use git calls rather than the API, as
		// afaics, there's no high-level calls to clone or pull a repo. One
		// needs to recurse through all things and fetch them manually. Lame.

		name := *(repo.Name)
		isFork := *(repo.Fork)
		if isFork {
			if !*flagForks && name != *flagRepo {
				if *flagVerbose {
					log.Printf("%v is a fork, skipping it.", name)
				}
				continue
			}
		}

		// If repo does not exist, clone it.
		if _, err := os.Stat(name); err != nil {
			if !os.IsNotExist(err) {
				log.Fatal(err)
			}
			if *flagVerbose {
				log.Printf("cloning %v", name)
			}
			cmd := exec.Command("git", "clone", *(repo.CloneURL))
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Fatalf("%v, %v", err, string(out))
			}
			if *flagVerbose {
				log.Printf("%v", string(out))
			}
			continue
		}

		// otherwise pull.
		if err := os.Chdir(name); err != nil {
			log.Fatal(err)
		}
		if *flagVerbose {
			log.Printf("pulling %v", name)
		}
		cmd := exec.Command("git", "pull")
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("%v, %v", err, string(out))
		}
		if *flagVerbose {
			log.Printf("%v", string(out))
		}
		if err := os.Chdir(cwd); err != nil {
			log.Fatal(err)
		}
	}
}
