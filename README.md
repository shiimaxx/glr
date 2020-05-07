# glr

`glr` is a tool for creating [GitLab Release](https://docs.gitlab.com/ee/user/project/releases/).


## Example


```
$ glr -upload ./dist v0.1.0
[Created release]
Title: v0.1.0
Release assets:
--> glr_v0.1.0_darwin_amd64.zip: https://gitlab.com/shiimaxx/glr-demo/uploads/.../glr_v0.1.0_darwin_amd64.zip
--> glr_v0.1.0_linux_amd64.tar.gz: https://gitlab.com/shiimaxx/glr-demo/uploads/.../glr_v0.1.0_linux_amd64.tar.gz
--> glr_v0.1.0_windows_amd64.zip: https://gitlab.com/shiimaxx/glr-demo/uploads/.../glr_v0.1.0_windows_amd64.zip
```

## Usage

You can run `glr` as following after changing the current directory to GitLab project root and setting GitLab Token to environment variable named `GITLAB_TOKEN`.

```
$ glr [options] TAG
```

`TAG` is a git tag. You must be specified a git tag in an argument.

When including assets in the release, use `-upload` option for uploading assets, or use `-asset-name` and `--asset-url` options for specifying any link.


## GitLab API Endpoint

Default GitLab API Endpoint is `https://gitlab.com/api/v4/`. You can change it via `GITLAB_API`.

```
export GITLAB_API=https://gitlab.example.com/api/v4/
```


## Release assets links

GitLab Release has two type of release assets that are **Souce code** and **Links**.

`glr` support creating asset links. Each link has properties that name and URL.

URL is a link to actual asset file like

- Built artifacts in GitLab CI
- External file
- Uploaded file in the project

If you want to create assets links for exists assets like build artifacts and external file, you can use `-asset-name` and `--asset-url` options for specifying assets URL.

Also, if you use `-upload` option, can uploading local assets at the same time of creating a release and create assets links for that.

See also [GitLab Docs - GitLab Release](https://docs.gitlab.com/ee/user/project/releases/index.html).

