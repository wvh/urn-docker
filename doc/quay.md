# RedHat Quay repository

Docker builds multi-layered packages of applications. These images have to be uploaded to a repository so they can be _pulled_ and used in services.

The most common public repository for docker images is [Docker Hub](https://hub.docker.com/); another one is RedHat's [Quay container repository](https://quay.io/).

This document lists the necessary steps to configure and push images to RedHat's Quay repository.

## log into the RedHat Quay website

Open the [Quay repository website](https://quay.io/) and log in, creating an account if necessary. You can log in with your Github account, for instance.

## create robot account

Next, in account settings, create a robot account so if the credentials get stolen, the attackers at least won't have full control over the Quay account.

## copy robot token

Optionally, copy the robot access token into the docker configuration file `~/.docker/config.json` if you want to log in without typing a password. This might be handy if you plan on updating images regularly from a secure machine.

```json
{
	"auths": {
		"quay.io": {
			"auth": "TOKEN",
		}
	},
	[...]
}
```
## sign into Quay with docker

```shell
docker login quay.io
```

If you skipped the token step above, you will have to supply either your own or the robot's login and password arguments to `docker login`:

```shell
docker login -u account+robot -p <ROBOT_TOKEN>
```

## create a new repository

On the Quay website, create a new repository for each image.

## give the robot account access

In Repository Settings, give the robot account `write` access to the newly created repository.

## push image to Quay repository

Now it should be possible to push images to the repository with docker.

For instance, for an already built image `image-hash-or-name`, add an appropriate tag so docker knows where to upload the image to, and then run `docker push` on that tag:

```shell
docker tag image-hash-or-name quay.io/account/repo:VERSION
docker push quay.io/account/repo:VERSION
```

Here `account` is your Quay account, `repo` is the repository you created on the website, and `version` is a tag either based on the git version or image hash to help identity what is actually running later on.

It makes sense to automate this step by means of a shell script or Makefile, taking the version information from the versioning system.
