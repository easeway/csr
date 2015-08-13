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
             |--install
             |   |--install1.sh
             |   |--install2.sh
             |--uninstall
                 |--uninstall1.sh
                 |--uninstall2.sh
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

    sync [NAME...] [--setup|-s]
        Synchronize named (or all if names not specified) local scripting
        repositories.
        When --setup is specified, run install scripts even there's no updates

    clean
        Remove all symbolic links.

    version
        Display version information.
```

## Supported Platforms

- Linux
- MacOS 10.8 and later

## License

MIT
