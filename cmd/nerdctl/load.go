/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/images/archive"
	"github.com/containerd/containerd/platforms"
	"github.com/urfave/cli/v2"
)

var loadCommand = &cli.Command{
	Name:        "load",
	Usage:       "Load an image from a tar archive or STDIN",
	Description: "Supports both Docker Image Spec v1.2 and OCI Image Spec v1.0.",
	Action:      loadAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "input",
			Aliases: []string{"i"},
			Usage:   "Read from tar archive file, instead of STDIN",
		},
		&cli.BoolFlag{
			Name:  "all-platforms",
			Usage: "Imports content for all platforms",
		},
	},
}

func loadAction(clicontext *cli.Context) error {
	in := clicontext.App.Reader
	if input := clicontext.String("input"); input != "" {
		f, err := os.Open(input)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}
	return loadImage(in, clicontext, false)
}

func loadImage(in io.Reader, clicontext *cli.Context, quiet bool) error {
	client, ctx, cancel, err := newClient(clicontext, containerd.WithDefaultPlatform(platforms.DefaultStrict()))
	if err != nil {
		return err
	}
	defer cancel()

	sn := clicontext.String("snapshotter")
	imgs, err := client.Import(ctx, in, containerd.WithDigestRef(archive.DigestTranslator(sn)), containerd.WithSkipDigestRef(func(name string) bool { return name != "" }), containerd.WithAllPlatforms(clicontext.Bool("all-platforms")))
	if err != nil {
		return err
	}
	for _, img := range imgs {
		image := containerd.NewImage(client, img)

		// TODO: Show unpack status
		if !quiet {
			fmt.Fprintf(clicontext.App.Writer, "unpacking %s (%s)...", img.Name, img.Target.Digest)
		}
		err = image.Unpack(ctx, sn)
		if err != nil {
			return err
		}
		if quiet {
			fmt.Fprintln(clicontext.App.Writer, img.Target.Digest)
		} else {
			fmt.Fprintf(clicontext.App.Writer, "done\n")
		}
	}

	return nil
}
