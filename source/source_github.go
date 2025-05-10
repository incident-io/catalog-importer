package source

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
	kitlog "github.com/go-kit/kit/log"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/go-github/v52/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

type SourceGitHub struct {
	Repos []string   `json:"repos"`
	Files []string   `json:"files"`
	Token Credential `json:"token"`
	// ExcludeArchived, if true, will cause the source to exclude files from
	// repositories that have been archived. Defaults to false (archived repositories are included).
	ExcludeArchived bool `json:"excludeArchived,omitempty"`
}

func (s SourceGitHub) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Repos, validation.Each(
			validation.Match(regexp.MustCompile("^[^/]/.+$")).
				Error("repos must be of the form owner/repo, or owner/* for matching all repos under that organization"),
		)),
	)
}

func (s SourceGitHub) String() string {
	return fmt.Sprintf("github (repos=%s files=%s)", s.Repos, s.Files)
}

func (s SourceGitHub) Load(ctx context.Context, logger kitlog.Logger, _ *http.Client) ([]*SourceEntry, error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(s.Token)},
	)))

	type Target struct {
		Owner   string // e.g. incident-io
		Repo    string // e.g. catalog-importer
		Ref     string // e.g. main or master
		Matches []*github.TreeEntry
	}

	// Use this whenever we're modifying structures that are race unsafe.
	var mu sync.Mutex
	synchronise := func(do func()) {
		defer mu.Unlock()
		mu.Lock()

		do()
	}

	// Expand any repo wildcards so we have a full list of repos for each owner we want to
	// scan.
	targets := []*Target{}
	addTarget := func(logger kitlog.Logger, repo *github.Repository) {
		synchronise(func() {
			if repo.GetArchived() && s.ExcludeArchived {
				logger.Log("msg", "skipping archived repo, excludeArchived is true",
					"owner", repo.Owner.GetLogin(), "repo", repo.GetName())
				return
			}

			target := &Target{
				Owner: repo.Owner.GetLogin(),
				Repo:  repo.GetName(),
				Ref:   repo.GetDefaultBranch(),
			}

			logger.Log("msg", "found GitHub repo",
				"owner", target.Owner, "repo", target.Repo, "ref", target.Ref)
			targets = append(targets, target)
		})
	}

	{
		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(10)

		for _, providedRepo := range s.Repos {
			components := strings.SplitN(providedRepo, "/", 2)
			if len(components) != 2 {
				return nil, fmt.Errorf("invalid format for repo must be owner/repo but got '%s'", providedRepo)
			}

			g.Go(func() error {
				owner, repoNameOrWildcard := components[0], components[1]

				// We've found an owner/* repo, so we must find all the repos under this organisation.
				if repoNameOrWildcard == "*" {
					logger.Log("msg", "found repo wildcard, resolving repo list", "owner", owner)
					opts := &github.RepositoryListByOrgOptions{
						ListOptions: github.ListOptions{PerPage: 100},
					}
					for {
						logger.Log("msg", "paging for repos...")
						page, resp, err := client.Repositories.ListByOrg(ctx, owner, opts)
						if err != nil {
							return errors.Wrap(err, fmt.Sprintf("listing GitHub repos for organization '%s'", owner))
						}
						for _, repo := range page {
							addTarget(logger, repo)
						}
						if resp.NextPage == 0 {
							break
						}
						opts.Page = resp.NextPage
					}
				} else {
					// The repo is specified, so we just need to resolve it so we can fetch the
					// default branch.
					resolved, _, err := client.Repositories.Get(ctx, owner, repoNameOrWildcard)
					if err != nil {
						return errors.Wrap(err, fmt.Sprintf("accessing '%s/%s'", owner, repoNameOrWildcard))
					}

					addTarget(logger, resolved)
				}

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, err
		}
	}

	{
		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(10)

		for idx := range targets {
			target := targets[idx]

			g.Go(func() error {
				logger.Log("msg", "listing GitHub tree",
					"owner", target.Owner, "repo", target.Repo, "ref", target.Ref)
				tree, _, err := client.Git.GetTree(ctx, target.Owner, target.Repo, target.Ref, true)
				if err != nil {
					if repositoryEmpty(err) {
						logger.Log("msg", "GitHub repository is empty, skipping",
							"owner", target.Owner, "repo", target.Repo, "ref", target.Ref)
						return nil
					}
					return errors.Wrap(err, fmt.Sprintf("getting tree for '%s/%s' at ref %s", target.Owner, target.Repo, target.Ref))
				}

				for _, pattern := range s.Files {
					for _, treeEntry := range tree.Entries {
						if treeEntry.GetType() != "blob" {
							continue // we're only interested in files
						}

						match, err := doublestar.Match(pattern, treeEntry.GetPath())
						if err != nil {
							return errors.Wrap(err, "matching file pattern")
						}

						if match {
							synchronise(func() {
								target.Matches = append(target.Matches, treeEntry)
							})
						}
					}
				}

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, err
		}
	}

	entries := []*SourceEntry{}
	{
		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(10)

		for idx := range targets {
			target := targets[idx]

			for jdx := range target.Matches {
				match := target.Matches[jdx]

				g.Go(func() error {
					blob, _, err := client.Git.GetBlob(ctx, target.Owner, target.Repo, match.GetSHA())
					if err != nil {
						return errors.Wrap(err,
							fmt.Sprintf("getting blob for '%s' from repo '%s/%s' at SHA %s", match.GetPath(), target.Owner, target.Repo, target.Ref))
					}

					if blob.GetEncoding() != "base64" {
						logger.Log("msg", "found matching GitHub file but incorrect encoding meant it wasn't processed",
							"owner", target.Owner, "repo", target.Repo, "ref", target.Ref, "path", match.GetPath())
						return nil
					}

					data, err := base64.StdEncoding.DecodeString(blob.GetContent())
					if err != nil {
						return errors.Wrap(err,
							fmt.Sprintf("decoding base64 blob for '%s' from repo '%s/%s' at SHA %s", match.GetPath(), target.Owner, target.Repo, target.Ref))
					}

					synchronise(func() {
						logger.Log("msg", "found matching GitHub file",
							"owner", target.Owner, "repo", target.Repo, "ref", target.Ref, "path", match.GetPath())
						entries = append(entries, &SourceEntry{
							Origin:   fmt.Sprintf("github (repo=%s/%s path=%s)", target.Owner, target.Repo, match.GetPath()),
							Filename: match.GetPath(),
							Content:  []byte(data),
						})
					})

					return nil
				})
			}
		}

		if err := g.Wait(); err != nil {
			return nil, err
		}
	}

	return entries, nil
}

func repositoryEmpty(err error) bool {
	if err == nil {
		return false
	}

	// Check for existing 409 "Git Repository is empty" error.
	// This is one way an empty or uninitialized repository might manifest.
	if strings.Contains(err.Error(), "409 Git Repository is empty") {
		return true
	}

	// Check for 404 Not Found from GetTree, which GitHub API returns for an empty repository.
	// Ref: https://docs.github.com/en/rest/git/trees?apiVersion=2022-11-28#get-a-tree
	// "If the tree SHA is not provided, the default branch will be used.
	//  If the repository is empty, a 404 will be returned."
	if ghErr, ok := err.(*github.ErrorResponse); ok {
		if ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound {
			// This indicates the tree object itself wasn't found. For a GetTree operation
			// on a repository confirmed to exist, this is the documented behavior for an empty repository.
			return true
		}
	}

	return false
}
