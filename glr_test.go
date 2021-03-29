package main

import (
	"os"
	"strings"
	"testing"

	"github.com/xanzy/go-gitlab"
)

// testProjectID is project id of https://gitlab.com/shiimaxx/glr-demo
var testProjectID = 18630472

func testGlr(t *testing.T) *glr {
	t.Helper()

	svc, err := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"), gitlab.WithBaseURL(defaultGitLabAPIEndpoint))
	if err != nil {
		t.Fatal("create gitlab client failed: ", err)
	}
	return &glr{
		baseURL: defaultGitLabEndpoint,
		projectPath: "shiimaxx/glr-demo",
		tag: "v1.2.3",
		title: "test release",
		description: "test release description",

		svc: svc,
	}
}

func Test_getProject(t *testing.T) {
	g := testGlr(t)

	proj, err := g.getProject("shiimaxx/glr-demo")
	if err != nil {
		t.Fatal("get project failed: ", err)
	}

	if got, want := proj.Name, "glr-demo"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func Test_uploadFiles(t *testing.T) {
	g := testGlr(t)
	files := []string{
	 	"testdata/glr-demo_v1.2.3_linux_amd64",
	  	"testdata/glr-demo_v1.2.3_darwin_amd64",
	}

	assets, err := g.uploadFiles(testProjectID, files)
	if err != nil {
		t.Fatal("upload files failed: ", err)
	}

	if got, want := len(assets), 2; got != want {
		t.Fatalf("upload files number: got %d, want, %d", got, want)
	}

	for i := range assets {
		if got, want := assets[i].name, strings.Split(files[i], "/")[1]; got != want {
			t.Fatalf("asset file name: got %s, want, %s", got, want)
		}
	}
}

func Test_createRelease(t *testing.T) {
	g := testGlr(t)

	if err := g.createRelease(testProjectID, nil); err != nil {
		t.Fatal("create release failed: ", err)
	}

	defer g.deleteRelease(testProjectID)
}

func Test_getLocalAssets(t *testing.T) {
	assets, err := getLocalAssets("testdata")
	if err != nil {
		t.Fatal("get local assets failed: ", err)
	}

	if got, want := len(assets), 2; got != want {
		t.Fatalf("upload files number: got %d, want, %d", got, want)
	}

	files := []string{
	  	"testdata/glr-demo_v1.2.3_darwin_amd64",
	 	"testdata/glr-demo_v1.2.3_linux_amd64",
	}
	for i := range assets {
		if got, want := assets[i], files[i]; got != want {
			t.Fatalf("asset file name: got %s, want, %s", got, want)
		}
	}
}
