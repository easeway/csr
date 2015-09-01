# Common Scripting Repository

## What is this?

This is a simple command line tool `csr` for Linux/Mac to manage common scripts.

Thinking of working in a team with many enthusiastic engineers who keep contributing useful tool scripts for other members continously.
It's always difficult and painful to setup the scripts correctly in individual's working environment.
Here's how CSR helps with a simple and unified experience across the whole team.

1. Setup a git repository for submitting the tool scripts
2. For each member, do a one time initial setup of CSR: `csr add http://host/scripts-repo.git`

From there, the contributors keep pushing scripts to this git repository, and others simply use `csr sync` to get the fresh scripts.
All scripts submitted in the git repository will be directly accessible from command line.

For example, a contributor added a shell script called "show-me-the-magic".
After all other members typed `csr sync`, they can use `show-me-the-magic` directly from commandline.

## How to build

```bash
export GOPATH=$(pwd)/vendor
go get -d .
go build csr.go

# Recommended installation
sudo mv csr /usr/local/bin/
sudo chown 0:0 /usr/local/bin/csr
sudo chmod a+rsx /usr/local/bin/csr # the s bit allow csr maintain symlinks when invoked with a non-privileged user
```

Don't worry about `s` bit set on `csr` executable.
It only elevate to root privilege when installing symbolic links.
It runs as current user invoking the actual scripts.

## How it works?

`csr` is a simple binary built from `csr.go`,
and usually people put it in `/usr/local/bin`
and put a sticky bit on it `chmod +s /usr/local/bin/csr` to allow `csr` manage symbolic links
under `/usr/local/bin` which is read-only to non-root users, and `sudo` can be avoid.

`csr` clones git repository into `~/.csr/repos/repo-name`, and tries to invoke install scripts each time after synchronized with remote repository.

`csr` scans all scripts under `bin` folder in the repository and creates symbolic links in `/usr/local/bin` with exactly the same name and pointing to `/usr/local/bin/csr`.
Similarly to `busybox`, `csr` relies on `args[0]` to determine which command to invoke and invokes the actual script in local git repository.

## Repository structure

```
/+
 |--suites/
     |--<name>/
         |--bin/
         |   |--command1
         |   |--command2
         |--setup/
         |   |--install
         |   |   |--install1.sh
         |   |   |--install2.sh
         |   |--uninstall
         |       |--uninstall1.sh
         |       |--uninstall2.sh
         |--docs/
             |--command1.md
             |--command2.md
```

A repository contains multiple `suites` each containing a set of commands in `bin` folder, and also a set of `install` and `uninstall` scripts under `setup` folder.
After update (`csr sync` or `csr add`), `install` scripts are invoked one-by-one. Before remove (`csr rm`), `uninstall` scripts are invoked one-by-one.

## Usage

```
Usage: <Command> [Options] [Arguments...]

    add GIT-REPO-URL [NAME]
        Install a scripting repository to local (~/.csr/repos). NAME is
        derived from GIT-REPO-URL if not specified. The operation will fail
        if NAME already exists.

    rm NAME
        Remove a scripting repository from local (~/.csr/repos).

    list
        List currently installed scripting repositories and all commands.

    help NAME
        View command document.
        The markdown document is rendered and piped to "less".

    sync [NAME...] [--setup|-s] [--local|-l]
        Synchronize named (or all if names not specified) local scripting
        repositories.
        When --setup is specified, run install scripts even there's no updates
        When --local is specified, skip fetching changes from remote, and only
        synchronize symbolic links.

    clean
        Remove all symbolic links.

    version
        Display version information.
```

## Environment for scripts

The following environment variables are pre-set for all binaries invoked by `csr`:

- `CSR_OS`: host OS type, can be `linux`, `darwin`
- `CSR_ARCH`: host CPU architecture, can be `386`, `amd64`
- `CSR_BIN_DIR`: directory for installing symbolic links, default to `/usr/local/bin`
- `CSR_BIN`: full path to `csr` binary, default to `$CSR_BIN_DIR/csr`
- `CSR_REPOS_BASE`: path to directory containing all repositories, default to `$HOME/.csr/repos`
- `CSR_REPO_NAME`: repository name containing invoked binary
- `CSR_REPO_DIR`: path to the repository containing invoked binary, it's `$CSR_REPOS_BASE/$CSR_REPO_NAME`
- `CSR_SUITE_NAME`: name of the suite containing invoked binary
- `CSR_SUITE_DIR`: path to the suite containing invoked binary, it's `$CSR_REPO_DIR/suites/$CSR_SUITE_NAME`
- `CSR_COMMAND`: the actual command invoked

## Documenting the commands

The sub folder `docs` under a suite is used for looking up document of a command.
The file name must be the same as the command with extension `.md`.

## Supported Platforms

- Linux
- MacOS 10.8 and later

## License

MIT
