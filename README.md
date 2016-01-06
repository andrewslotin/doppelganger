Doppelganger
============

A tool to mirror GitHub repositories.

Installation
------------

Create new [personal access token](http://github.com/settings/tokens) with following permissions:

* `public_repo`
* `write:repo_hook`

Docker:

```git
docker pull andrewslotin/doppelganger
docker run -d --name doppelganger -e DOPPELGANGER_GITHUB_TOKEN=<YOUR_PERSONAL_ACCESS_TOKEN> -v /home/git:/var/mirrors -p 8081:8081 andrewslotin/doppelganger
```

See [Docker Hub Page](https://hub.docker.com/r/andrewslotin/doppelganger/) for details.

Mirroring Private Repositories
------------------------------

To mirror private repositories your personal access token should have `repo` permission. To clone repositories you need to put into your 
container a private SSH key that was added to your GitHub account. It might be a good idea to use a separate key for Doppelganger.

```bash
docker cp ~/.ssh/doppelganger_dsa doppelganger:/root/.ssh/id_dsa
```