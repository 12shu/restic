package main

import (
	restic "github.com/restic/restic/internal"
	"github.com/spf13/cobra"
)

var cmdCache = &cobra.Command{
	Use:   "cache [name]",
	Short: "update the cache migration",
	Long: `
The "cache" command updates the cache.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCache(cacheOptions, globalOptions, args)
	},
}

// CacheOptions bundles all options for the 'check' command.
type CacheOptions struct {
}

var cacheOptions CacheOptions

func init() {
	cmdRoot.AddCommand(cmdCache)
}

func runCache(opts CacheOptions, gopts GlobalOptions, args []string) error {
	repo, err := OpenRepository(gopts)
	if err != nil {
		return err
	}

	if err := repo.LoadIndex(gopts.ctx); err != nil {
		return err
	}

	lock, err := lockRepo(repo)
	defer unlockRepo(lock)
	if err != nil {
		return err
	}

	types := []restic.FileType{
		restic.IndexFile,
		restic.SnapshotFile,
	}

	for _, tpe := range types {
		Printf("updating cache for %v:\n", tpe)
		for id := range repo.List(gopts.ctx, tpe) {
			h := restic.Handle{Type: tpe, Name: id.String()}
			Printf("  index %v\n", h)
			rd, err := repo.Backend().Load(gopts.ctx, h, 0, 0)
			if err != nil {
				return err
			}

			err = repo.Cache.Save(h, rd)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
