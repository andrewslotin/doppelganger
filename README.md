Doppelganger
============

A tool to create and maintain mirrors of GitHub repositories. Once the repostiory is mirrored a 
webhook is set up, so next time you push to GitHub `master` mirror will be updated. You may also trigger an update 
manually by clicking "Synchronize repository".

Why Would I Need It?
--------------------

Well, if you ask this question then most likely you won't. But next time some Chinese website 
[decides to host their assets on GitHub](http://arstechnica.com/security/2015/03/massive-denial-of-service-attack-on-github-tied-to-chinese-government/) you know you've been warned.

Installation
------------

Create new [personal access token](http://github.com/settings/tokens) with following permissions:

* `public_repo`
* `write:repo_hook`

Make sure you have `git` installed.

Download the [latest release](https://github.com/andrewslotin/doppelganger/releases), extract it and then run:

```bash
DOPPELGANGER_GITHUB_TOKEN=<YOUR_PERSONAL_ACCESS_TOKEN> ./doppelganger 
```

This will run a server listening on `0.0.0.0:8081` and all mirrored repositories will go to `./mirror`. 

Run `./doppelganger --help` to learn how to override these defaults.

### Build From Source

```bash
# Download and compile
go get github.com/andrewslotin/doppelganger
# Current working dir should have templates/ and assets/ folders
cd $GOPATH/src/github.com/andrewslotin/doppelganger
# Run it
DOPPELGANGER_GITHUB_TOKEN=<YOUR_PERSONAL_ACCESS_TOKEN> $GOPATH/bin/doppelganger
```

### Docker

```bash
docker pull andrewslotin/doppelganger
docker run -d --name doppelganger -e DOPPELGANGER_GITHUB_TOKEN=<YOUR_PERSONAL_ACCESS_TOKEN> -v /home/git:/var/mirrors -p 8081:8081 andrewslotin/doppelganger
```

See [Docker Hub Page](https://hub.docker.com/r/andrewslotin/doppelganger/) for details.

Usage
-----

Mirrors can be used just like any other git remote. You can even push your changes there directly, but note that they will be discarded next time someone pushes 
to GitHub `master`.

Following exaples assume you have `git` user set up on your mirror server with `HOME` set to Doppelganger mirror directory.

Set up a new local copy of `github.com/example/project` from mirror:

```bash
git clone git@<doppelganger-host>:example/project
```

Use mirror as a second remote in already existing repository:

```bash
git add remote mirror git@<doppelganger-host>:example/project

# Make your local master using mirror as an upstream:
git branch --unset-upstream master
git branch --set-upstream master mirror/master
git reset master mirror/master

# Switch back to GitHub
git branch --unset-upstream master
git branch --set-upstream master origin/master
```

Mirroring Private Repositories
------------------------------

To mirror private repositories your personal access token should have `repo` permission. To clone repositories you need to put into your 
container a private SSH key that was added to your GitHub account. It might be a good idea to use a separate key for Doppelganger.

```bash
docker cp ~/.ssh/doppelganger_dsa doppelganger:/root/.ssh/id_dsa
```