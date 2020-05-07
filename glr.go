package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/tcnksm/go-gitconfig"
	gitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/sync/errgroup"
)

const (
	defaultGitLabEndpoint    = "https://gitlab.com"
	defaultGitLabAPIEndpoint = defaultGitLabEndpoint + "/api/v4"

	exitCodeOK    = 0
	exitCodeError = 10 + iota
	exitCodeParseError
	exitCodeInvalidResponseCode
)

var (
	repoURLPattern = regexp.MustCompile(`([^/:]+)/([^/]+?)(?:\.git)?$`)
)

type glr struct {
	baseURL     string
	projectPath string
	tag         string
	title       string
	description string
	assetName   string
	assetURL    string

	svc *gitlab.Client
}

type asset struct {
	name string
	url  string
}

type cli struct {
	outStream, errStream io.Writer
}

func (g *glr) getProject(path string) (*gitlab.Project, error) {
	project, res, err := g.svc.Projects.GetProject(path, &gitlab.GetProjectOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "get project failed")
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("api request failed: invalid status: %d", res.StatusCode)
	}

	return project, nil
}

func (g *glr) uploadFiles(pid int, files []string) ([]*asset, error) {
	eg := errgroup.Group{}

	assets := make([]*asset, len(files))

	for i, f := range files {
		i := i
		f := f
		eg.Go(func() error {
			uploadedFile, _, err := g.svc.Projects.UploadFile(pid, f)
			if err != nil {
				return err
			}
			assets[i] = &asset{
				name: uploadedFile.Alt,
				url:  fmt.Sprintf("%s/%s%s", g.baseURL, g.projectPath, uploadedFile.URL),
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return assets, nil
}

func (g *glr) deleteRelease(pid int) error {
	rel, _, err := g.svc.Releases.GetRelease(pid, g.tag)
	if err != nil {
		return errors.Wrap(err, "get release failed")
	}

	for _, l := range rel.Assets.Links {
		if _, _, err := g.svc.ReleaseLinks.DeleteReleaseLink(pid, g.tag, l.ID); err != nil {
			return errors.Wrap(err, "delete release link failed")
		}
	}

	if _, _, err := g.svc.Releases.DeleteRelease(pid, g.tag); err != nil {
		return errors.Wrap(err, "delete release failed")
	}

	return nil
}

func (g *glr) createRelease(pid int, assets []*asset) error {
	var links []*gitlab.ReleaseAssetLink

	for _, a := range assets {
		links = append(links, &gitlab.ReleaseAssetLink{
			Name: a.name,
			URL:  a.url,
		})
	}

	if _, _, err := g.svc.Releases.CreateRelease(pid, &gitlab.CreateReleaseOptions{
		Name:        gitlab.String(g.title),
		TagName:     gitlab.String(g.tag),
		Description: gitlab.String(g.description),
		Assets:      &gitlab.ReleaseAssets{Links: links},
	}); err != nil {
		return errors.Wrap(err, "create release failed")
	}

	return nil
}

func getLocalAssets(root string) ([]string, error) {
	p, err := filepath.Abs(root)
	if err != nil {
		return nil, errors.Wrap(err, "Get absolute path failed")
	}

	f, err := os.Stat(p)
	if err != nil {
		return nil, errors.Wrap(err, "Get file stat failed")
	}

	var localArtifacts []string
	if f.IsDir() {
		filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			// exclude directory and hidden file
			if !info.IsDir() && !strings.HasPrefix(p, ".") {
				localArtifacts = append(localArtifacts, p)
			}
			return nil
		})
	} else {
		localArtifacts = []string{root}
	}

	return localArtifacts, nil
}

func (c *cli) run(args []string) int {
	var (
		tag string

		name      string
		body      string
		assetURL  string
		assetName string
		upload    string
		replace   bool

		versionFlag bool
	)

	flags := flag.NewFlagSet(appName, flag.ExitOnError)
	flags.Usage = func() { c.usage() }
	flags.SetOutput(c.outStream)
	flags.StringVar(&name, "name", "", "")
	flags.StringVar(&name, "n", "", "")
	flags.StringVar(&body, "body", "", "")
	flags.StringVar(&body, "b", "", "")
	flags.StringVar(&assetURL, "asset-url", "", "")
	flags.StringVar(&assetName, "asset-name", "", "")
	flags.StringVar(&upload, "upload", "", "")
	flags.BoolVar(&replace, "replace", false, "")
	flags.BoolVar(&versionFlag, "version", false, "")
	if err := flags.Parse((args[1:])); err != nil {
		fmt.Fprint(c.errStream, "flag parse failed: ", err)
		return exitCodeParseError
	}

	if versionFlag {
		fmt.Fprintf(c.outStream, "%s version %s\n", appName, version)
		return exitCodeOK
	}

	if len(flags.Args()) < 1 {
		fmt.Fprintln(c.outStream, "Missing arguments")
		return exitCodeParseError
	}

	tag = flags.Arg(0)

	if name == "" {
		name = tag
	}

	if body == "" {
		body = tag
	}

	origin, err := gitconfig.OriginURL()
	if err != nil {
		fmt.Fprintln(c.errStream, "Fetch origin url failed: ", err)
		return exitCodeError
	}
	matches := repoURLPattern.FindStringSubmatch(origin)

	endpoint := os.Getenv("GITLAB_API")
	if endpoint == "" {
		endpoint = defaultGitLabAPIEndpoint
	}

	svc, err := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"), gitlab.WithBaseURL(endpoint))
	if err != nil {
		fmt.Fprintln(c.errStream, "Create gitlab client failed: ", err)
	}

	g := glr{
		baseURL:     defaultGitLabEndpoint,
		projectPath: fmt.Sprintf("%s/%s", matches[1], matches[2]),
		tag:         tag,
		title:       name,
		description: body,

		svc: svc,
	}

	proj, err := g.getProject(g.projectPath)
	if err != nil {
		fmt.Fprintln(c.errStream, "Get project failed: ", err)
		return exitCodeError
	}

	var assets []*asset
	if upload != "" {
		localAssets, err := getLocalAssets(upload)

		if err != nil {
			fmt.Fprintln(c.errStream, "Get local assets failed: ", err)
			return exitCodeError
		}

		assets, err = g.uploadFiles(proj.ID, localAssets)
		if err != nil {
			fmt.Fprintln(c.errStream, "Upload files failed: ", err)
			return exitCodeError
		}
	}

	if assetName != "" && assetURL != "" {
		assets = append(assets, &asset{name: assetName, url: assetURL})
	}

	if replace {
		if err := g.deleteRelease(proj.ID); err != nil {
			fmt.Fprint(c.errStream, "Delete release failed: ", err)
			return exitCodeError
		}
	}

	if err := g.createRelease(proj.ID, assets); err != nil {
		fmt.Fprintln(c.errStream, "Create release failed: ", err)
		return exitCodeError
	}
	fmt.Fprintln(c.outStream, "[Created release]")
	fmt.Fprintf(c.outStream, "Title: %s\n", g.title)
	fmt.Fprintln(c.outStream, "Release assets:")
	if g.assetName != "" && g.assetURL != "" {
		fmt.Fprintf(c.outStream, "--> %s: %s\n", g.assetName, g.assetURL)
	}
	if len(assets) > 0 {
		for _, a := range assets {
			fmt.Fprintf(c.outStream, "--> %s: %s\n", a.name, a.url)
		}
	}

	return exitCodeOK
}

func (c *cli) usage() {
	fmt.Fprintf(c.errStream, `Usage: glr [options] TAG

glr is a tool for creating GitLab Release.

Options:
	-name, -n		Set release title. Default is TAG.
	-body, -b		Set description for release. Default is TAG.
	-asset-url		Set asset url.
	-asset-name		Set asset name.
	-upload			Set local asset path.
	-replace		Replace when GitLab Release is already exists.
	-version		Print version.

Argument:
	TAG			Set exists tag name.
`)
}
