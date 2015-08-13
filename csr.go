package main

import (
    "fmt"
    "io"
    "os"
    "os/exec"
    "os/user"
    "path/filepath"
    "sort"
    "strings"
    "syscall"
)

const (
    OriginalProgram = "csr"
    DataDir = ".csr"
    RepoDir = "repos"

    Version = "1.0.0"
    VersionInfo = "Common Scripting Repository v" + Version

    usageText = VersionInfo + `

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
    `
)

var (
    reposDir    = filepath.Join("~", DataDir, RepoDir)
    destBinDir  = "/usr/local/bin"
    destOrigBin = filepath.Join(destBinDir, OriginalProgram)
)

type localRepo struct {
    name string
}

type repoCommand struct {
    suite   string
    command string
    path    string
}

func allLocalRepos() (repos []*localRepo) {
    repoNames := make([]string, 0)
    d, err := os.Open(filepath.Join(reposDir))
    if err != nil {
        return
    }
    defer d.Close()
    for {
        if infos, err := d.Readdir(1); err != nil {
            break
        } else if infos[0].IsDir() {
            repoNames = append(repoNames, infos[0].Name())
        }
    }
    sort.Strings(repoNames)
    repos = make([]*localRepo, 0, len(repoNames))
    for _, name := range repoNames {
        repos = append(repos, &localRepo{name})
    }
    return
}

func (r *localRepo) exists() bool {
    _, err := os.Lstat(r.path())
    return err == nil
}

func (r *localRepo) path() string {
    return filepath.Join(reposDir, r.name)
}

func (r *localRepo) clone(url string) error {
    r.log("CREATE", "Clone from " + url)
    if err := os.MkdirAll(reposDir, 0777); err != nil {
        if !os.IsExist(err) {
            return err
        }
    }
    _, err := git(reposDir, "clone", url, r.path())
    return err
}

func (r *localRepo) update() (bool, error) {
    r.log("UPDATE", "")
    if commit, err := git(r.path(), "log", "-1", "--format=%H"); err != nil {
        return false, err
    } else if _, err := git(r.path(), "pull"); err != nil {
        return false, err
    } else if newCommit, err := git(r.path(), "log", "-1", "--format=%H"); err != nil {
        return false, err
    } else {
        return newCommit != commit, nil
    }
}

func (r *localRepo) setup(mode string) error {
    r.log("SETUP", mode)
    scripts, err := filepath.Glob(filepath.Join(r.path(), "suites", "*", "setup", mode, "*"))
    if err != nil {
        r.log("FAIL", err.Error())
        return err
    }
    sort.Strings(scripts)
    for _, script := range scripts {
        if !executable(script) {
            continue
        }
        if tokens := strings.Split(script, "/"); len(tokens) >= 4 {
            suite := tokens[len(tokens)-4]
            r.log("SETUP", strings.Join(tokens[len(tokens)-4:], "/"))
            cmd := exec.Command(script, mode)
            cmd.Stdin = os.Stdin
            cmd.Stdout = os.Stdout
            cmd.Stderr = os.Stderr
            cmd.Dir = filepath.Join(r.path(), "suites", suite)
            cmd.Env = r.prepareEnv(suite)
            if err := cmd.Run(); err != nil {
                return err
            }
        }
    }
    return nil
}

func (r *localRepo) remove() error {
    if err := r.setup("uninstall"); err != nil {
        return err
    }
    return os.RemoveAll(r.path())
}

func (r *localRepo) commands() []*repoCommand {
    cmds := make([]*repoCommand, 0)
    if scripts, err := filepath.Glob(filepath.Join(r.path(), "suites", "*", "bin", "*")); err == nil {
        for _, script := range scripts {
            if !executable(script) {
                continue
            }
            tokens := strings.Split(script, "/")
            if len(tokens) >= 4 {
                cmd := &repoCommand{
                    suite: tokens[len(tokens)-3],
                    command: tokens[len(tokens)-1],
                    path: script,
                }
                cmds = append(cmds, cmd)
            }
        }
    }
    return cmds
}

func (r *localRepo) exec(suite, cmd string, args []string) error {
    return syscall.Exec(filepath.Join(r.path(), "suites", suite, "bin", cmd), args, append(r.prepareEnv(suite), "CSR_COMMAND=" + cmd))
}

func (r *localRepo) prepareEnv(suite string) []string {
    return append(os.Environ(),
        "CSR_SUITE_DIR=" + filepath.Join(r.path(), "suites", suite),
        "CSR_REPO_NAME=" + r.name,
        "CSR_REPO_DIR=" + r.path())
}

func (r *localRepo) log(action string, msg string) {
    fmt.Fprintf(os.Stdout, "%s [%s] %s\n", action, r.name, msg)
}

func executable(script string) bool {
    if info, err := os.Stat(script); err != nil || info.IsDir() {
        return false
    } else if (info.Mode() & 0111) == 0 {
        return false
    }
    return true
}

func git(wd string, args ...string) (string, error) {
    cmd := exec.Command("git", args...)
    cmd.Stdin = os.Stdin
    cmd.Stderr = os.Stderr
    cmd.Dir = wd
    if out, err := cmd.Output(); err != nil {
        return "", err
    } else {
        return string(out), err
    }
}

func setuid() {
    if st, err := os.Stat(destBinDir); err == nil {
        ownerUid := st.Sys().(*syscall.Stat_t).Uid
        if currentUser, err := user.Current(); err == nil {
            if fmt.Sprintf("%v", ownerUid) != currentUser.Uid {
                syscall.Setuid(int(ownerUid))
            }
        }
    }
}

func linkedCmds(forInstall bool) ([]string, error) {
    cmds := make([]string, 0)
    d, err := os.Open(destBinDir)
    if err != nil {
        if os.IsNotExist(err) && forInstall {
            if err := os.MkdirAll(destBinDir, 0755); err != nil {
                fmt.Fprintf(os.Stderr, "Unable to create bin directory: %s: %v\n", destBinDir, err);
                return cmds, err
            }
            d, err = os.Open(destBinDir)
        }
    }
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to access bin directory: %s: %v\n", destBinDir, err);
        return cmds, err
    }
    defer d.Close()
    for {
        if info, err := d.Readdir(1); err != nil {
            if err == io.EOF {
                break
            }
            fmt.Fprintf(os.Stderr, "Scanning bin directory %s failed: %v\n", destBinDir, err)
            return cmds, err
        } else if info[0].IsDir() || (info[0].Mode() & os.ModeSymlink) == 0 {
            continue
        } else if name, err := os.Readlink(filepath.Join(destBinDir, info[0].Name())); err != nil {
            continue
        } else if name != OriginalProgram && name != destOrigBin {
            continue
        } else {
            cmds = append(cmds, info[0].Name())
        }
    }
    return cmds, nil
}

func installRepo(url, name string) {
    r := &localRepo{name}
    if r.exists() {
        fmt.Fprintf(os.Stderr, "Repository exists: %v\n", name)
        os.Exit(1)
    }

    if err := r.clone(url); err != nil {
        fmt.Fprintf(os.Stderr, "Clone repository failed: %v\n", err)
        os.Exit(1)
    }

    syncRepos(false, name, "--setup")
}

func uninstallRepo(name string) {
    r := &localRepo{name}
    if r.exists() {
        err := r.remove()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Remove repository %s failed: %v\n", r.name, err)
        }
        syncRepos(false)
        if err != nil {
            os.Exit(1)
        }
    }
}

func listRepoAndCommands() {
    repos := allLocalRepos()
    for _, r := range repos {
        fmt.Fprintf(os.Stdout, "[%s]\n", r.name)
        cmds := r.commands()
        for _, cmd := range cmds {
            fmt.Fprintf(os.Stdout, "    %s\n", cmd.command)
        }
    }
}

func syncRepos(update bool, names ...string) {
    allRepos := allLocalRepos()
    forceSetup := false
    selectedNames := make(map[string]bool)
    for _, name := range names {
        if name == "--setup" || name == "-s" {
            forceSetup = true
            continue
        }
        selectedNames[name] = true
    }

    cmdSrc := make(map[string][]string)
    fails := 0
    for _, r := range allRepos {
        if len(selectedNames) == 0 || selectedNames[r.name] {
            updated := false
            var err error = nil
            if update {
                if updated, err = r.update(); err != nil {
                    fmt.Fprintf(os.Stderr, "Unable to update repository %s: %v\n", r.name, err)
                    fails ++
                }
            }
            if err == nil && (forceSetup || updated) {
                if err = r.setup("install"); err != nil {
                    fmt.Fprintf(os.Stderr, "Setup repository %s failed: %v\n", r.name, err)
                    fails ++
                }
            }
        }
        cmds := r.commands()
        for _, cmd := range cmds {
            cmdSrc[cmd.command] = append(cmdSrc[cmd.command], r.name + "/" + cmd.suite)
        }
    }
    if fails > 0 {
        os.Exit(1)
    }

    existingCmds, err := linkedCmds(true)
    if err != nil {
        os.Exit(1)
    }
    sort.Strings(existingCmds)

    allCmds := make([]string, 0, len(cmdSrc))
    for cmd, src := range cmdSrc {
        if len(src) > 1 {
            fmt.Fprintf(os.Stderr, "WARNING: ambigous command: %s\n", cmd)
            for _, s := range src {
                fmt.Fprintf(os.Stderr, "    Defined in %s\n", s)
            }
        }
        allCmds = append(allCmds, cmd)
    }
    sort.Strings(allCmds)

    setuid()

    i := 0
    j := 0
    for i < len(allCmds) && j < len(existingCmds) {
        if allCmds[i] < existingCmds[j] {
            bin := filepath.Join(destBinDir, allCmds[i])
            fmt.Fprintf(os.Stdout, "+ %s\n", bin)
            if err := os.Symlink(OriginalProgram, bin); err != nil {
                fmt.Fprintf(os.Stderr, "Failed to create symlink: %s: %v\n", bin, err)
                fails ++
            }
            i ++
        } else if allCmds[i] > existingCmds[j] {
            bin := filepath.Join(destBinDir, existingCmds[j])
            fmt.Fprintf(os.Stdout, "- %s\n", bin)
            os.Remove(bin)
            j ++
        } else {
            fmt.Fprintf(os.Stdout, "* %s\n", filepath.Join(destBinDir, allCmds[i]))
            i ++
            j ++
        }
    }
    for i < len(allCmds) {
        bin := filepath.Join(destBinDir, allCmds[i])
        fmt.Fprintf(os.Stdout, "+ %s\n", bin)
        if err := os.Symlink(OriginalProgram, bin); err != nil {
            fmt.Fprintf(os.Stderr, "Failed to create symlink: %s: %v\n", bin, err)
            fails ++
        }
        i ++
    }
    for j < len(existingCmds) {
        bin := filepath.Join(destBinDir, existingCmds[j])
        fmt.Fprintf(os.Stdout, "- %s\n", bin)
        os.Remove(bin)
        j ++
    }
    if fails > 0 {
        os.Exit(1)
    }
}

func cleanSymlinks() {
    if cmds, err := linkedCmds(false); err != nil {
        if !os.IsNotExist(err) {
            os.Exit(1)
        }
    } else {
        setuid()
        for _, cmd := range cmds {
            bin := filepath.Join(destBinDir, cmd)
            fmt.Fprintf(os.Stdout, "- %s\n", bin)
            if err := os.Remove(bin); err != nil {
                fmt.Fprintf(os.Stderr, "Remove %s failed: %v\n", cmd, err)
            }
        }
    }
}

func showVersion() {
    fmt.Fprintf(os.Stdout, "%s\n", VersionInfo)
}

func showUsageAndExit() {
    fmt.Fprintln(os.Stderr, usageText)
    os.Exit(2)
}

func runOriginal() {
    if len(os.Args) <= 1 {
        showUsageAndExit()
    }

    switch (os.Args[1]) {
        case "add":
            if len(os.Args) < 3 {
                showUsageAndExit()
            } else {
                repoUrl := os.Args[2]
                repoName := ""
                if len(os.Args) > 3 {
                    repoName = os.Args[3]
                } else {
                    nameSrc := repoUrl
                    if pos := strings.LastIndex(nameSrc, "/"); pos >= 0 {
                        nameSrc = nameSrc[pos+1:]
                    }
                    if strings.HasSuffix(nameSrc, ".git") {
                        nameSrc = nameSrc[0:len(nameSrc) - 4]
                    }
                    repoName = nameSrc
                }
                installRepo(repoUrl, repoName)
            }
        case "rm":
            if len(os.Args) < 3 {
                showUsageAndExit()
            } else {
                uninstallRepo(os.Args[2])
            }
        case "list":
            listRepoAndCommands()
        case "sync":
            syncRepos(true, os.Args[2:]...)
        case "clean":
            cleanSymlinks()
        case "version":
            showVersion()
        default:
            showUsageAndExit()
    }
}

func runDelegation(cmd string, args []string) {
    files, _ := filepath.Glob(filepath.Join(reposDir, "*", "suites", "*", "bin", cmd))
    var lastErr error = nil
    for _, fn := range files {
        tokens := strings.Split(fn, "/")
        if len(tokens) < 5 {
            continue
        }
        suite := tokens[len(tokens)-3]
        r := &localRepo{tokens[len(tokens)-5]}
        if lastErr = r.exec(suite, cmd, args); lastErr == nil {
            break
        }
    }
    if lastErr != nil {
        fmt.Fprintf(os.Stderr, "Exec failed: %v\n", lastErr)
        os.Exit(128)
    } else {
        fmt.Fprintf(os.Stderr, "Command not found: %s\n", cmd)
        os.Exit(1)
    }
}

func main() {
    if u, err := user.Current(); err == nil {
        reposDir = filepath.Join(u.HomeDir, DataDir, RepoDir)
    }
    if v := os.Getenv("CSR_REPOS_BASE"); v != "" {
        reposDir = v
    } else {
        os.Setenv("CSR_REPOS_BASE", reposDir)
    }
    if v := os.Getenv("CSR_BIN_DIR"); v != "" {
        destBinDir = v
        destOrigBin = filepath.Join(destBinDir, OriginalProgram)
    } else {
        os.Setenv("CSR_BIN_DIR", destBinDir)
    }
    if v := os.Getenv("CSR_BIN"); v != "" {
        destOrigBin = v
    } else {
        os.Setenv("CSR_BIN", destOrigBin)
    }

    if base := filepath.Base(os.Args[0]); base == OriginalProgram {
        runOriginal()
    } else {
        runDelegation(base, os.Args)
    }
}
